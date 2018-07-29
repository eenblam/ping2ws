package main

import (
	"log"
	"net/http"

	"github.com/eenblam/ping2ws"
)

func main() {
	targets := []string{
		"8.8.8.8",
		"172.30.0.1",
		"127.0.0.1",
	}
	m := ping2ws.NewMonitor(targets)
	defer m.Stop()
	h := http.NewServeMux()
	h.HandleFunc("/monitor", m.PingHandler)
	h.Handle("/", http.FileServer(http.Dir("./static")))
	log.Fatal(http.ListenAndServeTLS(":8080", "server.crt", "server.key", h))
}
