package transport

type Config struct {
	Addr               string `yaml:"addr"`
	ReadTimeoutMillis  int    `yaml:"read_timeout_millis"`
	WriteTimeoutMillis int    `yaml:"write_timeout_millis"`
}

func withDefaultConf(conf *Config) error {
	if conf.Addr == "" {
		conf.Addr = ":8080"
	}

	if conf.ReadTimeoutMillis < 1 {
		conf.ReadTimeoutMillis = 5000
	}

	if conf.WriteTimeoutMillis < 1 {
		conf.WriteTimeoutMillis = 5000
	}

	return nil
}
