package openshift

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
	"k8s.io/client-go/util/homedir"
)

// CheckKubeConfigExist checks for existence of kubeconfig
func CheckKubeConfigExist() bool {

	var kubeconfig string

	if os.Getenv("KUBECONFIG") != "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	} else {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
			log.Printf("using default kubeconfig path %s", kubeconfig)
		} else {
			log.Printf("no KUBECONFIG provided and cannot fallback to default")
			return false
		}
	}

	if CheckPathExists(kubeconfig) {
		return true
	}

	return false
}

// CheckPathExists checks if a path exists or not
func CheckPathExists(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		// path to file does exist
		return true
	}
	log.Printf("path %s doesn't exist, skipping it", path)
	return false
}

// GetHostWithPort parses provided url and returns string formated as
// host:port even if port was not specifically specified in the origin url.
// If port is not specified, standart port corresponding to url schema is provided.
// example: for url https://example.com function will return "example.com:443"
//          for url https://example.com:8443 function will return "example:8443"
func GetHostWithPort(inputURL string) (string, error) {
	u, err := url.Parse(inputURL)
	if err != nil {
		return "", errors.Wrapf(err, "error while getting port for url %s ", inputURL)
	}

	port := u.Port()
	address := u.Host
	// if port is not specified try to detect it based on provided scheme
	if port == "" {
		portInt, err := net.LookupPort("tcp", u.Scheme)
		if err != nil {
			return "", errors.Wrapf(err, "error while getting port for url %s ", inputURL)
		}
		port = strconv.Itoa(portInt)
		address = fmt.Sprintf("%s:%s", u.Host, port)
	}
	return address, nil
}
