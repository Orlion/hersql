package main

import (
	"flag"
	"fmt"
	"github.com/Orlion/hersql/config"
	"github.com/Orlion/hersql/entrance"
	"github.com/Orlion/hersql/log"
	"os"
)

var configFile *string = flag.String("config", "", "hersql entrance config file")

func main() {
	flag.Parse()
	conf, err := config.ParseEntranceConfig(*configFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "configuration file parse error: "+err.Error())
		os.Exit(1)
	}

	log.Init(conf.Log)

	server := entrance.NewServer(conf.Server)
	if err = server.ListenAndServe(); err != nil {
		fmt.Fprintln(os.Stderr, "server error: "+err.Error())
		os.Exit(1)
	}
}
