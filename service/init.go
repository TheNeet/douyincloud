package service

import "os"

var hostname string

func Init() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		panic(err)
	}
	managerRun()
}
