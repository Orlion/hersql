package sidecar

import "errors"

type Config struct {
	Addr          string `yaml:"addr"`
	TransportAddr string `yaml:"transport_addr"`
}

func withDefaultConf(conf *Config) error {
	if conf == nil {
		return errors.New("server configuration cannot be empty")
	}

	if conf.Addr == "" {
		conf.Addr = "127.0.0.1:3306"
	}

	if conf.TransportAddr == "" {
		return errors.New("transport_addr configuration cannot be empty")
	}

	return nil
}
