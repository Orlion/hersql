package sidecar

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"

	"github.com/Orlion/hersql/transport"
)

func (c *Conn) transportConnect() error {
	form := url.Values{}
	form.Set("addr", c.dsn.Addr)
	form.Set("dbname", c.dsn.DBName)
	form.Set("user", c.dsn.User)
	form.Set("passwd", c.dsn.Passwd)
	form.Set("collation", strconv.FormatUint(uint64(c.collation), 10))

	body, err := c.callTransport("/connect", form)
	if err != nil {
		return err
	}

	response := new(transport.ConnectResponse)
	if err := json.Unmarshal(body, response); err != nil {
		return fmt.Errorf("transport response body unmarshal error: %w", err)
	}

	if !response.Success {
		return errors.New(response.Msg)
	}

	c.transportRunid = response.Data.Runid
	c.transportConnId = response.Data.ConnId

	return nil
}

func (c *Conn) transportDisconnect() error {
	form := url.Values{}
	form.Set("runid", c.transportRunid)
	form.Set("connId", strconv.FormatUint(c.transportConnId, 10))
	body, err := c.callTransport("/disconnect", form)
	if err != nil {
		return err
	}

	response := new(transport.Response)
	if err := json.Unmarshal(body, response); err != nil {
		return fmt.Errorf("transport response body unmarshal error: %w", err)
	}

	if !response.Success {
		return errors.New(response.Msg)
	}

	return nil
}

func (c *Conn) transport(data []byte) ([][]byte, error) {
	form := url.Values{}
	form.Set("runid", c.transportRunid)
	form.Set("connId", strconv.FormatUint(c.transportConnId, 10))
	form.Set("packet", string(data))
	body, err := c.callTransport("/transport", form)
	if err != nil {
		return nil, err
	}

	response := new(transport.TransportResponse)
	if err := json.Unmarshal(body, response); err != nil {
		return nil, fmt.Errorf("transport response body unmarshal error: %w", err)
	}

	if !response.Success {
		return nil, errors.New(response.Msg)
	}

	return response.Data, nil
}

func (c *Conn) callTransport(path string, form url.Values) ([]byte, error) {
	var (
		body []byte
	)
	resp, err := c.server.transportClient.PostForm(c.server.transportAddr+path, form)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
