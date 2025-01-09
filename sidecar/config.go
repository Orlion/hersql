package sidecar

import "errors"

type Config struct {
	Addr               string `yaml:"addr"`
	TransportAddr      string `yaml:"transport_addr"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
	Version            string `yaml:"version"`
}

func withDefaultConf(conf *Config) error {
	if conf == nil {
		return errors.New("server configuration cannot be empty")
	}

	if conf.Addr == "" {
		conf.Addr = DefaultServerAddr
	}

	if conf.TransportAddr == "" {
		return errors.New("transport_addr configuration cannot be empty")
	}

	if conf.Version == "" {
		conf.Version = DefaultServerVersion
	}

	return nil
}
