package exit

import "net/http"

func Serve(conf *Config) error {
	handler := NewHandler()
	http.HandleFunc("/connect", handler.HandleConnect)
	http.HandleFunc("/disconnect", handler.HandleDisconnect)
	http.HandleFunc("/transport", handler.HandleTransport)
	return http.ListenAndServe(conf.Addr, nil)
}
