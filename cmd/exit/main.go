package main

import (
	"flag"
	"fmt"
	"github.com/Orlion/hersql/config"
	"github.com/Orlion/hersql/exit"
	"github.com/Orlion/hersql/log"
	"os"
)

var configFile *string = flag.String("config", "", "hersql exit config file")

func main() {
	flag.Parse()
	conf, err := config.ParseExitConfig(*configFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "configuration file parse error: "+err.Error())
		os.Exit(1)
	}

	log.Init(conf.Log)

	if err = exit.Serve(conf.Server); err != nil {
		fmt.Fprintln(os.Stderr, "server error: "+err.Error())
		os.Exit(1)
	}
}
