package transport

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Success bool   `json:"status"`
	Msg     string `json:"msg"`
}

type ConnectResponse struct {
	Response
	Data *ConnectResponseData `json:"data"`
}

type ConnectResponseData struct {
	Runid  string `json:"runid"`
	ConnId uint64 `json:"conn_id"`
}

type TransportResponse struct {
	Response
	Data [][]byte `json:"data"`
}

func connectResponse(w http.ResponseWriter, runid string, connId uint64) {
	b, err := json.Marshal(&ConnectResponse{
		Response: Response{
			Success: true,
		},
		Data: &ConnectResponseData{
			Runid:  runid,
			ConnId: connId,
		},
	})
	if err != nil {
		return
	}

	w.Write(b)
}

func disconnectResponse(w http.ResponseWriter) {
	b, err := json.Marshal(&Response{
		Success: true,
	})
	if err != nil {
		return
	}

	w.Write(b)
}

func transportResponse(w http.ResponseWriter, data [][]byte) {
	b, err := json.Marshal(&TransportResponse{
		Response: Response{
			Success: true,
		},
		Data: data,
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
