package rtmp

import (
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"net/http/httputil"
	"time"
)

type HttpServe struct {
	hostPort string
}

func NewHttpServer(port string) {
	var hs HttpServe
	hs.hostPort = port
	go hs.httpServerStart()
}

func handler1(w http.ResponseWriter, r *http.Request) {
	fmt.Println("the listen port:80")
	b, _ := httputil.DumpRequest(r, true)
	fmt.Println("Request")
	fmt.Println(string(b))
	time.Sleep(100 * time.Second)
	w.Write([]byte("Hello, world!"))

}

func (self *HttpServe) httpServerStart() {
	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/test", handler1)
	r.HandleFunc("/{app}/{name:[A-Za-z0-9-_+]+}.flv",HDLHandler)
	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(self.hostPort, r))
}
