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

func withDefaultConf(config *Config) *Config {
	if config == nil {
		config = &Config{
			StdoutLevel: "info",
		}

		return config
	}

	if config.StdoutLevel == "" {
		config.StdoutLevel = "info"
	}

	if config.Level == "" {
		config.Level = "error"
	}

	return config
}
