package entrance

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"

	"github.com/Orlion/hersql/exit"
	"github.com/Orlion/hersql/log"
)

func (c *Conn) exitConnect() error {
	form := url.Values{}
	form.Set("addr", c.dsn.Addr)
	form.Set("dbname", c.dsn.DBName)
	form.Set("user", c.dsn.User)
	form.Set("passwd", c.dsn.Passwd)

	resp, err := c.callExit("/connect", form)
	if err != nil {
		return err
	}

	if !resp.Success {
		return errors.New(resp.Msg)
	}

	connectData := resp.Data.(*exit.ResponseConnectData)

	c.exitRunid = connectData.Runid
	c.exitConnId = connectData.ConnId

	return nil
}

func (c *Conn) callExit(path string, form url.Values) (*exit.Response, error) {
	var (
		body []byte
	)
	defer log.Debugf("%s callExit%s form: %s, resp.body: %s", c.name(), path, form.Encode(), string(body))
	resp, err := c.server.exitClient.PostForm(c.server.exitServerAddr+path, form)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	exitResponse := new(exit.Response)
	if err := json.Unmarshal(body, exitResponse); err != nil {
		return nil, fmt.Errorf("resp body json unmarshal error: %w", err)
	}

	return exitResponse, nil
}

func (c *Conn) exitDisconnect() error {
	form := url.Values{}
	form.Set("runid", c.exitRunid)
	form.Set("connId", strconv.FormatUint(c.exitConnId, 10))
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
	form.Set("runid", c.exitRunid)
	form.Set("connId", strconv.FormatUint(c.exitConnId, 10))
	form.Set("packet", string(data))
	resp, err := c.callExit("/transport", form)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, errors.New(resp.Msg)
	}

	return resp.Data.([]byte), nil
}
