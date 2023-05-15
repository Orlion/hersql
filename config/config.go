package config

import (
	"errors"
	"io/ioutil"

	"github.com/Orlion/hersql/entrance"
	"github.com/Orlion/hersql/exit"
	"github.com/Orlion/hersql/log"
	"gopkg.in/yaml.v3"
)

type EntranceConfig struct {
	Log    *log.Config      `yaml:"log"`
	Server *entrance.Config `yaml:"server"`
}

type ExitConfig struct {
	Log    *log.Config  `yaml:"log"`
	Server *exit.Config `yaml:"server"`
}

func ParseEntranceConfig(filename string) (conf *EntranceConfig, err error) {
	if filename == "" {
		err = errors.New("please enter a configuration file name")
		return
	}

	fileData, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	conf = new(EntranceConfig)
	err = yaml.Unmarshal(fileData, conf)
	if err != nil {
		return
	}

	return
}

func ParseExitConfig(filename string) (conf *ExitConfig, err error) {
	if filename == "" {
		err = errors.New("please enter a configuration file name")
		return
	}

	fileData, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	conf = new(ExitConfig)
	err = yaml.Unmarshal(fileData, conf)
	if err != nil {
		return
	}

	return
}
