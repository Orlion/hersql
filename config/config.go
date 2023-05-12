package config

import (
	"errors"
	"github.com/Orlion/hersql/entrance"
	"github.com/Orlion/hersql/exit"
	"github.com/Orlion/hersql/log"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

type EntranceConfig struct {
	Log    *log.Config      `yaml:"log"`
	Server *entrance.Config `yaml:"server"`
}

type ExitConfig struct {
	Log    *log.Config  `yaml:"log"`
	Server *exit.Config `yaml:"server"`
}

func ParseEntranceConfig(filename string) (config *EntranceConfig, err error) {
	if filename == "" {
		err = errors.New("please enter a configuration file name")
		return
	}

	fileData, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	config = new(EntranceConfig)
	err = yaml.Unmarshal(fileData, config)
	if err != nil {
		return
	}

	return
}

func ParseExitConfig(filename string) (config *ExitConfig, err error) {
	if filename == "" {
		err = errors.New("please enter a configuration file name")
		return
	}

	fileData, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	config = new(ExitConfig)
	err = yaml.Unmarshal(fileData, config)
	if err != nil {
		return
	}

	return
}
