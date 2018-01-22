package rtmp

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httputil"
	"time"
	"net/http/pprof"
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
	defer func(){
		self.done <- false
	}()
	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/test", handler1)
	r.HandleFunc("/{app}/{name:[A-Za-z0-9-_+]+}.flv",HDLHandler)
	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

	// Manually add support for paths linked to by index page at /debug/pprof/
	r.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	r.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	r.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	r.Handle("/debug/pprof/block", pprof.Handler("block"))
	var ln net.Listener
	if ln,err=self.socketListen(addr);err != nil{
		return  err
	}
	// Bind to a port and pass our router in
	Hserver := &http.Server{Addr: addr, Handler: r}
	return Hserver.Serve(ln)
}
