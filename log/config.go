package log

type Config struct {
	StdoutLevel string `yaml:"stdout_level"`
	Level       string `yaml:"level"`
	Filename    string `yaml:"filename"`
	MaxSize     int    `yaml:"maxsize"`
	MaxAge      int    `yaml:"maxage"`
	MaxBackups  int    `yaml:"maxbackups"`
	Compress    bool   `yaml:"compress"`
}

func withDefaultConf(conf *Config) *Config {
	if conf == nil {
		conf = &Config{
			StdoutLevel: "info",
		}

		return conf
	}

	if conf.StdoutLevel == "" {
		conf.StdoutLevel = "info"
	}

	if conf.Level == "" {
		conf.Level = "error"
	}

	return conf
}
