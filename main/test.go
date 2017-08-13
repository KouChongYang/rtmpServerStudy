package main

import "rtmpServerStudy/rtmp"

func main() {
	server := &rtmp.Server{}
	server.ListenAndServe()
}
