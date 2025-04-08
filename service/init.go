package service

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

const ByteHeaderLogID = "X-TT-LOGID"

var (
	hostname string
	envStr   string

	envHeaders = make(http.Header)
)

func Init() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		panic(err)
	}
	envStr = os.Getenv("X_TT_ENV")
	if len(envStr) != 0 {
		envHeaders.Set("X-TT-ENV", envStr)
		envHeaders.Set("X-USE-PPE", "1")
	}
	managerRun()
}

func CtxInfo(ctx *gin.Context, msg string, args ...interface{}) {
	fmt.Println("[Info]", ctx.GetHeader(ByteHeaderLogID), fmt.Sprintf(msg, args))
}

func CtxError(ctx *gin.Context, msg string, args ...interface{}) {
	fmt.Println("[Error]", ctx.GetHeader(ByteHeaderLogID), fmt.Sprintf(msg, args))
}
