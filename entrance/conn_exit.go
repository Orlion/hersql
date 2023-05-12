package entrance

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/Orlion/hersql/exit"
	"io"
	"net/url"
	"strconv"
)

func (c *Conn) exitConnect() error {
	form := url.Values{}
	form.Set("host", c.user)
	form.Set("db", c.db)
	form.Set("user", c.user)
	form.Set("password", c.user)

	resp, err := c.callExit("/connect", form)
	if err != nil {
		return err
	}

	if !resp.Success {
		return errors.New(resp.Msg)
	}

	c.exitConnId = binary.BigEndian.Uint64(resp.Data)

	return nil
}

func (c *Conn) callExit(path string, form url.Values) (*exit.Response, error) {
	resp, err := c.server.httpClient.PostForm(c.server.httpHost+path, form)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	exitResponse := new(exit.Response)
	if err := json.Unmarshal(body, exitResponse); err != nil {
		return nil, err
	}

	return exitResponse, nil
}

func (c *Conn) exitDisconnect() error {
	form := url.Values{}
	form.Set("connId", strconv.Itoa(int(c.exitConnId)))
	resp, err := c.callExit("/disconnect", form)
	if err != nil {
		return err
	}

	if !resp.Success {
		return errors.New(resp.Msg)
	}

	return nil
}

func (c *Conn) exitTransport(data []byte) ([]byte, error) {
	form := url.Values{}
	form.Set("data", string(data))
	resp, err := c.server.httpClient.PostForm(c.server.httpHost+"/transport", form)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return responseData, nil
}
