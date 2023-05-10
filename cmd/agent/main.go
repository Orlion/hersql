package main

import agent "github.com/Orlion/hersql/internal/agent"

func main() {
	server := agent.NewAgent()
	server.ListenAndServe()
}
