package rtmp
import (
	"net/http"
	"fmt"
	"github.com/gorilla/mux"
)



func HDLHandler(w http.ResponseWriter, r *http.Request){
	fmt.Println(r.URL.Path)

	name := mux.Vars(r)["name"]
	app := mux.Vars(r)["app"]
	fmt.Println(name,app)
}
