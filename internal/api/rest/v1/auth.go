package v1

import (
	"net/http"
)

func authenicate(w http.ResponseWriter, r *http.Request) {
	// TODO:
	// fmt.Printf("request RemoteAddr: %v\n", r.RemoteAddr)
	// contentLength, ok := r.Header["Content-Length"]
	// if ok {
	// 	fmt.Printf("header Content-Length: %v\n", contentLength)
	// }
	// authHeader, ok := r.Header["Authorization"]
	// if ok {
	// 	fmt.Printf("header Content-Length: %v\n", authHeader)
	// }
	// w.Write([]byte(fmt.Sprintf("got: %s\n", r.URL)))
	// w.WriteHeader(200)
	http.Error(w, "Not Implemented", http.StatusNotImplemented)
}
