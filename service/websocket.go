package service

import (
	"bytes"
	"context"
	"douyincloud-gin-demo/redis"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	WsActionWsPush    = "ws_push"
	WsActionHeartbeat = "heartbeat"
)

type UserMessage struct {
	Action    string `json:"action"`
	Body      string `json:"body,omitempty"`
	time      time.Time
	timeStamp string
}

type wsStruct struct {
	sessionID string
	ch        chan []*wsInfo
	ctx       context.Context
	cancel    context.CancelFunc
}

type wsInfo struct {
	Body    string      `json:"body"`
	Headers http.Header `json:"headers,omitempty"`
}

var wsSessionMap = sync.Map{}

func keyForWsInfos(sessionID string) string {
	return fmt.Sprintf("ws_infos_%s", sessionID)
}

func managerRun() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("err: %+v\npanic: %s\n", err, string(debug.Stack()))
			}
			managerRun()
		}()

		for {
			var delList []string
			wsSessionMap.Range(func(sessionID, value interface{}) bool {
				ws := value.(*wsStruct)
				_, err := redis.Get(ws.ctx, ws.sessionID)
				if errors.Is(err, redis.Nil) {
					ws.cancel()
					delList = append(delList, ws.sessionID)
					fmt.Printf("session_id: %s is not exist\n", ws.sessionID)
					return true
				}

				data, err := redis.Pop(ws.ctx, keyForWsInfos(ws.sessionID))
				if err != nil {
					fmt.Printf("redis.Pop failed, session_id: %s, err: %+v\n", ws.sessionID, err)
					return true
				}
				if len(data) != 0 {
					var infos []*wsInfo
					for _, v := range data {
						info := &wsInfo{}
						err = json.Unmarshal([]byte(v), info)
						if err != nil {
							fmt.Printf("json.Unmarshal failed, session_id: %s, data: %s, err: %+v\n", ws.sessionID, v, err)
							continue
						}
						infos = append(infos, info)
					}
					ws.ch <- infos
				}
				return true
			})
			for _, v := range delList {
				wsSessionMap.Delete(v)
			}
			time.Sleep(time.Millisecond * 10)
		}
	}()
}

func ConnectOrDisconnect(ctx *gin.Context) {
	switch ctx.GetHeader("X-tt-event-type") {
	case "connect":
		Connect(ctx)
	case "disconnect":
		Disconnect(ctx)
	default:
		fmt.Println("[Error] X-tt-event-type is not connect or disconnect")
		ctx.String(http.StatusBadRequest, "X-tt-event-type is not connect or disconnect")
		return
	}
}

func Connect(ctx *gin.Context) {
	sessionID := ctx.GetHeader("X-TT-SESSIONID")
	if sessionID == "" {
		fmt.Println("[Error] X-TT-SESSIONID is empty")
		ctx.String(http.StatusBadRequest, "X-TT-SESSIONID is empty")
		return
	}
	wsCtx, cancel := context.WithCancel(context.Background())
	ws := &wsStruct{sessionID, make(chan []*wsInfo), wsCtx, cancel}
	if err := redis.Set(ws.ctx, sessionID, "1", 0); err != nil {
		fmt.Printf("[Error] redis.Set failed, err: %+v\n", err)
		ctx.String(http.StatusBadRequest, "redis.Set failed")
		return
	}
	wsSessionMap.Store(sessionID, ws)
	wsRun(ws)
	fmt.Printf("connect success, session_id: %s, hostname: %s\n", sessionID, hostname)
	ctx.Status(http.StatusOK)
}

func wsRun(ws *wsStruct) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("[Panic] err: %+v\npanic: %s\n", err, string(debug.Stack()))
				wsRun(ws)
			}
		}()

		ticker := time.NewTicker(50 * time.Millisecond)
		for {
			select {
			case <-ws.ctx.Done():
				fmt.Println("wsRun done")
				return
			case wsInfos := <-ws.ch:
				if len(wsInfos) == 0 {
					continue
				}
				for _, info := range wsInfos {
					now := time.Now()
					timeStamp := now.Format(time.RFC3339)
					fmt.Printf("timeStamp: %s, sessionID: %s, wsRun receive wsInfo, body: %s, header: %+v\n", timeStamp, ws.sessionID, info.Body, info.Headers)
					msg := &UserMessage{}
					err := json.Unmarshal([]byte(info.Body), msg)
					if err != nil {
						fmt.Printf("[Error] timeStamp: %s, unmarshal failed, err: %+v", timeStamp, err)
						continue
					}
					msg.time = now
					msg.timeStamp = timeStamp
					wsRunAction(ws.sessionID, msg)
				}
			case <-ticker.C:
			}
		}
	}()
}

func wsRunAction(sessionID string, msg *UserMessage) {
	var err error
	switch msg.Action {
	case WsActionWsPush:
		err = wsRunActionWsPush(sessionID, msg)
	case WsActionHeartbeat:
		fmt.Printf("收到心跳：%s\n", msg.Body)
	default:
		fmt.Printf("time_stamp: %s, action: %s", msg.timeStamp, msg.Action)
	}
	if err != nil {
		fmt.Printf("[Error] time_stamp: %s, err: %+v", msg.timeStamp, err)
	}
}

func wsRunActionWsPush(sessionID string, msg *UserMessage) error {
	body := bytes.NewReader([]byte(`{"msg": "ws_push test"}`))
	req, err := http.NewRequest(http.MethodPost, "https://ws-push.dyc.ivolces.com/ws/push_data", body)
	if err != nil {
		return err
	}
	req.Header.Set("X-TT-WS-SESSIONIDS", fmt.Sprintf("[%s]", sessionID))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Printf("timeStamp: %s, http code: %d, resp body: %s, header: %+v", msg.timeStamp, resp.StatusCode, string(respBody), resp.Header)
	return nil
}

func Disconnect(ctx *gin.Context) {
	if ctx.GetHeader("X-tt-event-type") != "disconnect" {
		fmt.Println("[Error] X-tt-event-type is not disconnect")
		ctx.String(http.StatusBadRequest, "X-tt-event-type is not disconnect")
		return
	}
	sessionID := ctx.GetHeader("X-TT-SESSIONID")
	if sessionID == "" {
		fmt.Println("[Error] X-TT-SESSIONID is empty")
		ctx.String(http.StatusBadRequest, "X-TT-SESSIONID is empty")
		return
	}
	if err := redis.Del(ctx, sessionID); err != nil {
		fmt.Printf("[Error] redis.Del failed, err: %+v\n", err)
		ctx.String(http.StatusBadRequest, "redis.Del failed")
		return
	}
	data, ok := wsSessionMap.Load(sessionID)
	if ok {
		ws, ok := data.(*wsStruct)
		if !ok {
			fmt.Println("[Error] session type error")
			ctx.String(http.StatusBadRequest, "session type error")
			return
		}
		ws.cancel()
		wsSessionMap.Delete(sessionID)
	}
	fmt.Println("disconnect success")
	ctx.Status(http.StatusOK)
}

func Uplink(ctx *gin.Context) {
	if ctx.GetHeader("X-tt-event-type") != "uplink" {
		fmt.Println("[Error] X-tt-event-type is not uplink")
		ctx.String(http.StatusBadRequest, "X-tt-event-type is not uplink")
		return
	}
	sessionID := ctx.GetHeader("X-TT-SESSIONID")
	if sessionID == "" {
		fmt.Println("[Error] X-TT-SESSIONID is empty")
		ctx.String(http.StatusBadRequest, "X-TT-SESSIONID is empty")
		return
	}
	body, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		fmt.Printf("[Error] read body failed, err: %+v\n", err)
		ctx.String(http.StatusBadRequest, "read body error")
		return
	}
	info := &wsInfo{
		Body:    string(body),
		Headers: ctx.Request.Header,
	}
	data, ok := wsSessionMap.Load(sessionID)
	if !ok {
		if _, err = redis.Get(ctx, sessionID); err != nil {
			fmt.Printf("[Error] redis.Get failed, err: %+v\n", err)
			ctx.String(http.StatusBadRequest, "session get failed")
			return
		}
		redisData, err := json.Marshal(info)
		if err != nil {
			fmt.Printf("[Error] marshal failed, err: %+v\n", err)
			ctx.String(http.StatusBadRequest, "marshal error")
			return
		}
		err = redis.Push(ctx, keyForWsInfos(sessionID), string(redisData))
		if err != nil {
			fmt.Printf("[Error] redis set failed, err: %+v\n", err)
			ctx.String(http.StatusBadRequest, "redis set error")
			return
		}
	} else {
		ws, ok := data.(*wsStruct)
		if !ok {
			fmt.Println("[Error] session type error")
			ctx.String(http.StatusBadRequest, "session type error")
			return
		}
		ws.ch <- []*wsInfo{info}
	}
	fmt.Println("uplink success")
	ctx.Status(http.StatusOK)
}
