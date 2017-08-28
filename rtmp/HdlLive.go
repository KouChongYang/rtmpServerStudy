package rtmp
import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"fmt"
)

type HttpServe struct {
	hostport string
}

func NewHttpServer (hport string) {
	var hs HttpServe
	hs.hostport = hport
	go hs.httpServerStart()
}

func HdlHandler(w http.ResponseWriter, r *http.Request){
	fmt.Println(r.URL.Path)
}

func (self *HttpServe) httpServerStart() {
	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("{name}.flv",HdlHandler)
	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(self.hostport, r))
}