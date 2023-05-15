package entrance

import "errors"

type Config struct {
	Addr                    string `yaml:"addr"`
	ExitServerAddr          string `yaml:"exit_server_addr"`
	ReadTimeoutMillis       int    `yaml:"read_timeout_millis"`
	WriteTimeoutMillis      int    `yaml:"write_timeout_millis"`
	ExitServerTimeoutMillis int    `yaml:"exit_server_timeout_millis"`
}

func withDefaultConf(conf *Config) error {
	if conf == nil {
		return errors.New("server configuration cannot be empty")
	}

	if conf.Addr == "" {
		conf.Addr = "127.0.0.1:3306"
	}

	if conf.ExitServerAddr == "" {
		return errors.New("exit_server_addr configuration cannot be empty")
	}

	if conf.ReadTimeoutMillis < 1 {
		conf.ReadTimeoutMillis = 5000
	}

	if conf.WriteTimeoutMillis < 1 {
		conf.WriteTimeoutMillis = 5000
	}

	if conf.ExitServerTimeoutMillis < 1 {
		conf.WriteTimeoutMillis = 5000
	}

	return nil
}
