package service

import (
	"net/http"
	"os"
)

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
