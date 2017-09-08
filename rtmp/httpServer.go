package rtmp

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httputil"
	"time"
	"net"
)


func handler1(w http.ResponseWriter, r *http.Request) {
	fmt.Println("the listen port:80")
	b, _ := httputil.DumpRequest(r, true)
	fmt.Println("Request")
	fmt.Println(string(b))
	time.Sleep(100 * time.Second)
	w.Write([]byte("Hello, world!"))

}

func (self *Server) httpServerStart(addr string)(err error) {
	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/test", handler1)
	r.HandleFunc("/{app}/{name:[A-Za-z0-9-_+]+}.flv",HDLHandler)
	var ln net.Listener
	if ln,err=self.socketListen(addr);err != nil{
		return  err
	}
	// Bind to a port and pass our router in
	Hserver := &http.Server{Addr: addr, Handler: r}
	return Hserver.Serve(ln)
}
