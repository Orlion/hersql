package config

import (
	"errors"
	"os"

	"github.com/Orlion/hersql/log"
	"github.com/Orlion/hersql/sidecar"
	"github.com/Orlion/hersql/transport"
	"gopkg.in/yaml.v3"
)

type SidecarConfig struct {
	Log    *log.Config     `yaml:"log"`
	Server *sidecar.Config `yaml:"server"`
}

type TransportConfig struct {
	Log    *log.Config       `yaml:"log"`
	Server *transport.Config `yaml:"server"`
}

func ParseSidecarConfig(filename string) (conf *SidecarConfig, err error) {
	if filename == "" {
		err = errors.New("please enter a configuration file name")
		return
	}

	fileData, err := os.ReadFile(filename)
	if err != nil {
		return
	}

	conf = new(SidecarConfig)
	err = yaml.Unmarshal(fileData, conf)
	if err != nil {
		return
	}

	return
}

func ParseTransportConfig(filename string) (conf *TransportConfig, err error) {
	if filename == "" {
		err = errors.New("please enter a configuration file name")
		return
	}

	fileData, err := os.ReadFile(filename)
	if err != nil {
		return
	}

	conf = new(TransportConfig)
	err = yaml.Unmarshal(fileData, conf)
	if err != nil {
		return
	}

	return
}
