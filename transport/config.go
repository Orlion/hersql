package transport

type Config struct {
	Addr string `yaml:"addr"`
}

func withDefaultConf(conf *Config) error {
	if conf.Addr == "" {
		conf.Addr = ":8080"
	}

	return nil
}
