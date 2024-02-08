package main

import (
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/deckhouse/virtualization-csi-driver/pkg/driver"
	"github.com/deckhouse/virtualization-csi-driver/pkg/host"
	"github.com/deckhouse/virtualization-csi-driver/pkg/logger"
)

func main() {
	hostCluster, err := host.NewClient()
	if err != nil {
		panic(err)
	}

	var endpoint string
	flag.StringVar(&endpoint, "endpoint", "", "CSI endpoint")
	var isDebugMode bool
	flag.BoolVar(&isDebugMode, "debug", false, "debug mode")
	flag.Parse()

	if endpoint == "" {
		panic(errors.New("CSI endpoint missed but required"))
	}

	var opts []logger.Option
	if isDebugMode {
		opts = append(opts, logger.NewDebugOption())
	}

	csi, err := driver.New(endpoint, hostCluster, logger.New(opts))
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

	csi.Stop()
}
