package openshift

import (
	"fmt"
	"net"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	// api clientsets

	// used to confirm if required operators are present on the cluster
	operatorsclientset "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/typed/operators/v1alpha1"

	_ "k8s.io/client-go/plugin/pkg/client/auth" // Required for Kube clusters which use auth plugins
)

const (
	// errorMsg is the message for user when invalid configuration error occurs
	errorMsg = `
please confirm if you have a configured openshift/kubernetes cluster.
Error: %w`
	defaultQPS   = 200
	defaultBurst = 200
)

// Client is a collection of fields used for client configuration and interaction
type Client struct {
	KubeClient       kubernetes.Interface
	KubeConfig       clientcmd.ClientConfig
	KubeClientConfig *rest.Config
	Namespace        string
	OperatorClient   *operatorsclientset.OperatorsV1alpha1Client
}

// New creates a new client
func New() (*Client, error) {
	return NewForConfig(nil)
}

// NewForConfig creates a new client with the provided configuration or initializes the configuration if none is provided
func NewForConfig(config clientcmd.ClientConfig) (client *Client, err error) {
	if config == nil {
		// initialize client-go clients
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		config = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	}

	client = new(Client)
	client.KubeConfig = config

	client.KubeClientConfig, err = client.KubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf(errorMsg, err)
	}

	// For the rest CLIENT, we set the QPS and Burst to high values so
	// we do not receive throttling error messages when using the REST client.
	// Inadvertently, this also increases the speed of which we use the REST client
	// to safe values without increased error / query information.
	// See issue: https://github.com/kubernetes/client-go/issues/610
	// and reference implementation: https://github.com/vmware-tanzu/tanzu-framework/pull/1656
	client.KubeClientConfig.QPS = defaultQPS
	client.KubeClientConfig.Burst = defaultBurst

	client.KubeClient, err = kubernetes.NewForConfig(client.KubeClientConfig)
	if err != nil {
		return nil, err
	}

	client.Namespace, _, err = client.KubeConfig.Namespace()
	if err != nil {
		return nil, err
	}

	client.OperatorClient, err = operatorsclientset.NewForConfig(client.KubeClientConfig)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// IsServerUp checks if a openshift cluster is up or not
func (c *Client) IsServerUp(timeout time.Duration) (bool, error) {
	config, err := c.KubeConfig.ClientConfig()
	if err != nil {

		return false, fmt.Errorf("unable to get server's address: %w", err)
	}

	server := config.Host
	address, err := GetHostWithPort(server)
	if err != nil {

		return false, fmt.Errorf("unable to parse url %s (%s)", server, err)
	}
	_, connectionError := net.DialTimeout("tcp", address, timeout)
	if connectionError != nil {
		return false, fmt.Errorf("unable to parse url %s (%s)", server, err)
	}

	return true, fmt.Errorf("unable to parse url %s (%s)", server, err)
}
