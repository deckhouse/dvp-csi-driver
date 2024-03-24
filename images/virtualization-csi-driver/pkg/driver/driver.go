package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"

	"github.com/deckhouse/virtualization-csi-driver/pkg/host"
	mounter "github.com/deckhouse/virtualization-csi-driver/pkg/mounter"
)

type Driver struct {
	nodeName         string
	csiEndpoint      string
	livenessEndpoint string

	hostCluster *host.Client
	grpc        *grpc.Server
	http        *http.Server
	mounter     *mounter.Mounter

	logger *slog.Logger
}

// New returns a CSI plugin that contains the necessary gRPC
// interfaces to interact with Kubernetes over unix domain sockets for
// managaing  disks
func New(csiEndpoint, livenessEndpoint string, hostCluster *host.Client, logger *slog.Logger) (*Driver, error) {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		return nil, errors.New("node name env not found")
	}

	logger = logger.WithGroup("driver").With("host-id", nodeName)

	return &Driver{
		nodeName:         nodeName,
		csiEndpoint:      csiEndpoint,
		livenessEndpoint: livenessEndpoint,
		hostCluster:      hostCluster,
		mounter:          mounter.New(logger),
		logger:           logger,
	}, nil
}

func (d *Driver) Start() error {
	d.logger.Info("Start driver")

	err := d.startCSIEndpoint()
	if err != nil {
		return err
	}

	if d.livenessEndpoint != "" {
		err = d.startLivenessEndpoint()
		if err != nil {
			return err
		}
	}

	d.logger.Info("Driver started")

	return nil
}

func (d *Driver) Stop() error {
	d.logger.Info("Stop driver")

	d.grpc.GracefulStop()

	if d.http != nil {
		err := d.http.Shutdown(context.Background())
		if err != nil {
			return err
		}
	}

	d.logger.Info("Driver stopped")

	return nil
}

func (d *Driver) startCSIEndpoint() error {
	u, err := url.Parse(d.csiEndpoint)
	if err != nil {
		return fmt.Errorf("unable to parse address: %w", err)
	}

	grpcAddr := path.Join(u.Host, filepath.FromSlash(u.Path))
	if u.Host == "" {
		grpcAddr = filepath.FromSlash(u.Path)
	}

	// CSI plugins talk only over UNIX sockets currently
	if u.Scheme != "unix" {
		return fmt.Errorf("currently only unix domain sockets are supported, have: %s", u.Scheme)
	}

	// remove the socket if it's already there. This can happen if we
	// deploy a new version and the socket was created from the old running
	// plugin.
	err = os.Remove(grpcAddr)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove unix domain socket file %s, error: %w", grpcAddr, err)
	}

	grpcListener, err := net.Listen(u.Scheme, grpcAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	d.grpc = grpc.NewServer(grpc.UnaryInterceptor(d.logInterceptor))
	csi.RegisterIdentityServer(d.grpc, d)
	csi.RegisterControllerServer(d.grpc, d)
	csi.RegisterNodeServer(d.grpc, d)

	go func() {
		err := d.grpc.Serve(grpcListener)
		if err != nil {
			panic(err)
		}
	}()

	return nil
}

func (d *Driver) startLivenessEndpoint() error {
	httpListener, err := net.Listen("tcp", d.livenessEndpoint)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	go func() {
		err := d.http.Serve(httpListener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	d.http = &http.Server{
		Handler: mux,
	}

	return nil
}

func (d *Driver) logInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	d.logger.Info("Got request "+info.FullMethod, "req", string(data))

	res, err := handler(ctx, req)
	if err != nil {
		d.logger.Error("Request failed "+info.FullMethod, "err", err)
	} else {
		data, err = json.Marshal(res)
		if err != nil {
			return nil, err
		}
		d.logger.Info("Request processed "+info.FullMethod, "res", string(data))
	}

	return res, err
}
