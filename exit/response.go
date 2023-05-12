package exit

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Success bool   `json:"status""`
	Msg     string `json:"msg"`
	Data    []byte `json:"data"`
}

func responseSuccess(w http.ResponseWriter, data []byte) {
	b, err := json.Marshal(&Response{
		Success: true,
		Data:    data,
	})
	if err != nil {
		return
	}

	w.Write(b)
}

func responseFail(w http.ResponseWriter, msg string) {
	b, err := json.Marshal(&Response{
		Msg: msg,
	})
	if err != nil {
		return
	}

	w.Write(b)
}
