package service

import (
	"context"
	"douyincloud-gin-demo/redis"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

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

func managerRun() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("err: %+v\npanic: %s\n", err, string(debug.Stack()))
			}
			managerRun()
		}()

		for {
			wsSessionMap.Range(func(sessionID, value interface{}) bool {
				ws := value.(*wsStruct)
				data, err := redis.Pop(ws.ctx, ws.sessionID)
				if err != nil {
					fmt.Printf("redis.Pop failed, session_id: %s, err: %+v", ws.sessionID, err)
					return true
				}
				if len(data) != 0 {
					var infos []*wsInfo
					for _, v := range data {
						info := &wsInfo{}
						err = json.Unmarshal([]byte(v), info)
						if err != nil {
							fmt.Printf("json.Unmarshal failed, session_id: %s, data: %s, err: %+v", ws.sessionID, v, err)
							continue
						}
						infos = append(infos, info)
					}
					ws.ch <- infos
				}
				return true
			})
			time.Sleep(time.Millisecond * 10)
		}
	}()
}

func Connect(ctx *gin.Context) {
	if ctx.GetHeader("X-tt-event-type") != "connect" {
		fmt.Println("X-tt-event-type is not connect")
		ctx.String(http.StatusBadRequest, "X-tt-event-type is not connect")
		return
	}
	sessionID := ctx.GetHeader("X-TT-SESSIONID")
	if sessionID == "" {
		fmt.Println("X-TT-SESSIONID is empty")
		ctx.String(http.StatusBadRequest, "X-TT-SESSIONID is empty")
		return
	}
	wsCtx, cancel := context.WithCancel(context.Background())
	ws := &wsStruct{sessionID, make(chan []*wsInfo), wsCtx, cancel}
	wsSessionMap.Store(sessionID, ws)
	wsRun(ws)
	fmt.Printf("connect success, session_id: %s, hostname: %s\n", sessionID, hostname)
	ctx.Status(http.StatusOK)
}

func wsRun(ws *wsStruct) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("err: %+v\npanic: %s\n", err, string(debug.Stack()))
			}
			wsRun(ws)
		}()

		ticker := time.NewTicker(50 * time.Millisecond)
		for {
			select {
			case <-ws.ctx.Done():
				fmt.Println("wsRun done")
				return
			case data := <-ws.ch:
				fmt.Println("wsRun receive data", data)
			case <-ticker.C:
			}
		}
	}()
}

func Disconnect(ctx *gin.Context) {
	if ctx.GetHeader("X-tt-event-type") != "disconnect" {
		fmt.Println("X-tt-event-type is not disconnect")
		ctx.String(http.StatusBadRequest, "X-tt-event-type is not disconnect")
		return
	}
	sessionID := ctx.GetHeader("X-TT-SESSIONID")
	if sessionID == "" {
		fmt.Println("X-TT-SESSIONID is empty")
		ctx.String(http.StatusBadRequest, "X-TT-SESSIONID is empty")
		return
	}
	data, ok := wsSessionMap.Load(sessionID)
	if !ok {
		fmt.Println("session not found")
		ctx.String(http.StatusBadRequest, "session not found")
		return
	}
	ws, ok := data.(*wsStruct)
	if !ok {
		fmt.Println("session type error")
		ctx.String(http.StatusBadRequest, "session type error")
		return
	}
	ws.cancel()
	wsSessionMap.Delete(sessionID)
	fmt.Println("disconnect success")
	ctx.Status(http.StatusOK)
}

func Uplink(ctx *gin.Context) {
	if ctx.GetHeader("X-tt-event-type") != "uplink" {
		fmt.Println("X-tt-event-type is not uplink")
		ctx.String(http.StatusBadRequest, "X-tt-event-type is not uplink")
		return
	}
	sessionID := ctx.GetHeader("X-TT-SESSIONID")
	if sessionID == "" {
		fmt.Println("X-TT-SESSIONID is empty")
		ctx.String(http.StatusBadRequest, "X-TT-SESSIONID is empty")
		return
	}
	body, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		fmt.Println("read body error", err)
		ctx.String(http.StatusBadRequest, "read body error")
		return
	}
	info := &wsInfo{
		Body:    string(body),
		Headers: ctx.Request.Header,
	}
	data, ok := wsSessionMap.Load(sessionID)
	if !ok {
		redisData, err := json.Marshal(info)
		if err != nil {
			fmt.Println("marshal error", err)
			ctx.String(http.StatusBadRequest, "marshal error")
			return
		}
		err = redis.Push(ctx, sessionID, string(redisData))
		if err != nil {
			fmt.Println("redis set error", err)
			ctx.String(http.StatusBadRequest, "redis set error")
			return
		}
	} else {
		ws, ok := data.(*wsStruct)
		if !ok {
			fmt.Println("session type error")
			ctx.String(http.StatusBadRequest, "session type error")
			return
		}
		ws.ch <- []*wsInfo{info}
	}
	fmt.Println("uplink success")
	ctx.Status(http.StatusOK)
}
