package sidecar

import "errors"

type Config struct {
	Addr                   string `yaml:"addr"`
	TransportAddr          string `yaml:"transport_addr"`
	ReadTimeoutMillis      int    `yaml:"read_timeout_millis"`
	WriteTimeoutMillis     int    `yaml:"write_timeout_millis"`
	TransportTimeoutMillis int    `yaml:"transport_timeout_millis"`
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

	if conf.ReadTimeoutMillis < 1 {
		conf.ReadTimeoutMillis = 5000
	}

	if conf.WriteTimeoutMillis < 1 {
		conf.WriteTimeoutMillis = 5000
	}

	if conf.TransportTimeoutMillis < 1 {
		conf.TransportTimeoutMillis = 5000
	}

	return nil
}
