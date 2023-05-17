package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Orlion/hersql/config"
	"github.com/Orlion/hersql/log"
	"github.com/Orlion/hersql/sidecar"
)

var configFile *string = flag.String("conf", "", "hersql sidecar config file")

func main() {
	flag.Parse()
	conf, err := config.ParseSidecarConfig(*configFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "configuration file parse error: "+err.Error())
		os.Exit(1)
	}

	log.Init(conf.Log)

	srv, err := sidecar.NewServer(conf.Server)
	if err != nil {
		fmt.Fprintln(os.Stderr, "server error: "+err.Error())
		os.Exit(1)
	}

	go func() {
		if err = srv.ListenAndServe(); err != nil {
			fmt.Fprintln(os.Stderr, "server error: "+err.Error())
			os.Exit(1)
		}
	}()

	waitGracefulStop(srv)
}

func waitGracefulStop(srv *sidecar.Server) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		s := <-c
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			log.Infof("received signal: %s will stop...", s.String())
			ctx, _ := context.WithTimeout(context.Background(), 3000*time.Millisecond)
			srv.Shutdown(ctx)
			log.Shutdown()
			time.Sleep(1 * time.Second)
			return
		case syscall.SIGHUP:
		default:
		}
	}
}
