package host

import (
	"encoding/base64"
	"errors"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/deckhouse/virtualization/api/v1alpha2"
)

type Client struct {
	crClient  client.Client
	namespace string
}

func NewClient() (*Client, error) {
	kubeconfig := os.Getenv("HOST_KUBECONFIG")
	if kubeconfig == "" {
		return nil, errors.New("kubeconfig env not found")
	}

	hostNamespace := os.Getenv("HOST_NAMESPACE")
	if hostNamespace == "" {
		return nil, errors.New("host namespace env not found")
	}

	kubeconfigBase64, err := base64.StdEncoding.DecodeString(kubeconfig)
	if err != nil {
		return nil, err
	}

	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfigBase64)
	if err != nil {
		return nil, err
	}

	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	scheme := runtime.NewScheme()
	err = v1alpha2.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	crClient, err := client.New(config, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		crClient:  crClient,
		namespace: hostNamespace,
	}, nil
}
