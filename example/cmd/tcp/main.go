package main

import (
	"log"
	"net/http"

	"github.com/eenblam/ping2ws"
)

func main() {
	targets := []string{
		"127.0.0.1:21",
		"127.0.0.1:22",
		"127.0.0.1:53",
		"127.0.0.1:80",
		// This will be UP since we're serving on it,
		// but it will also generate a lot of logs from the webserver
		// http: TLS handshake error from 127.0.0.1:<port>: EOF
		"127.0.0.1:8080",
	}
	m := ping2ws.NewMonitorTCP(targets)
	defer m.Stop()
	h := http.NewServeMux()
	h.HandleFunc("/monitor", m.HandleWS)
	h.Handle("/", http.FileServer(http.Dir("./static")))
	log.Fatal(http.ListenAndServeTLS(":8080", "server.crt", "server.key", h))
}
