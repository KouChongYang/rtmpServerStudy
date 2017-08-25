package main
import "fmt"
import "rtmpServerStudy/rtmp"

func main() {
	var p *int
	a:=5
	p = &a
	var q *int
	q = p
	p=nil
	
	fmt.Println(q)
	server := &rtmp.Server{}
	server.ListenAndServe()
}
