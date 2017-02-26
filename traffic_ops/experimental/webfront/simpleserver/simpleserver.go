package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

var Logger *log.Logger

func hello(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Hitting "+os.Args[1]+" with "+r.URL.Path+"\n")
}

func main() {
	Logger = log.New(os.Stdout, " ", log.Ldate|log.Ltime|log.Lshortfile)
	
	http.HandleFunc("/", hello)

	// Make sure you have the server.pem and server.key file. To gen self signed:
	// openssl genrsa -out server.key 2048
	// openssl req -new -x509 -key server.key -out server.pem -days 3650
	Logger.Println(http.ListenAndServeTLS(":"+os.Args[1], "server.pem", "server.key", nil))
}
