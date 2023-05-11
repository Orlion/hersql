package main

import (
	"github.com/Orlion/hersql/internal/agents"
	"net/http"
)

func main() {
	http.HandleFunc("/conn", agents.HandleConn)
	http.HandleFunc("/conn", agents.HandleConn)
	http.ListenAndServe(":8009", nil)
}
