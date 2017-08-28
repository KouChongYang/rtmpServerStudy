package main

import (
	"rtmpServerStudy/rtmp"
	"runtime"
	"fmt"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU() - 1)
	go func() {
		fmt.Println(http.ListenAndServe(":6060", nil))
	}()
	server := &rtmp.Server{}
	server.ListenAndServe()
	rtmp.NewHttpServer("0.0.0.0:801")
}
