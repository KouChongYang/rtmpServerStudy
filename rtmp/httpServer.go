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
	port string
}

func NewHttpServer(port string) {
	var hs HttpServe
	hs.port = port
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
	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(self.port, r))
}
