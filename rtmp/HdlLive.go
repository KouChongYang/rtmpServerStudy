package rtmp
import (
	"net/http"
	"fmt"
)



func HdlHandler(w http.ResponseWriter, r *http.Request){
	fmt.Println(r.URL.Path)
}
