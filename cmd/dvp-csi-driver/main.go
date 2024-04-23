package main

import (
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/deckhouse/dvp-csi-driver/internal/driver"
	"github.com/deckhouse/dvp-csi-driver/internal/host"
	"github.com/deckhouse/dvp-csi-driver/internal/logger"
)

func main() {
	hostCluster, err := host.NewClient()
	if err != nil {
		panic(err)
	}

	var csiEndpoint string
	flag.StringVar(&csiEndpoint, "csi-endpoint", "", "CSI endpoint")
	var livenessEndpoint string
	flag.StringVar(&livenessEndpoint, "liveness-endpoint", "", "Liveness endpoint")
	var isDebugMode bool
	flag.BoolVar(&isDebugMode, "debug", false, "debug mode")
	flag.Parse()

	if csiEndpoint == "" {
		panic(errors.New("CSI endpoint missed but required"))
	}

	var opts []logger.Option
	if isDebugMode {
		opts = append(opts, logger.NewDebugOption())
	}

	csi, err := driver.New(csiEndpoint, livenessEndpoint, hostCluster, logger.New(opts))
	if err != nil {
		panic(err)
	}

	err = csi.Start()
	if err != nil {
		panic(err)
	}

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)

	<-exit

	err = csi.Stop()
	if err != nil {
		panic(err)
	}
}
