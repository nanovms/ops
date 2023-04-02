//go:build ibm || !onlyprovider

package ibm

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/nanovms/ops/types"
)

// Objects represent storage specific information for cloud object
// storage.
type Objects struct {
	token string
}

// CopyToBucket copies archive to bucket
func (s *Objects) CopyToBucket(config *types.Config, archPath string) error {
	zone := config.CloudConfig.Zone

	region := extractRegionFromZone(zone)

	bucket := config.CloudConfig.BucketName

	baseName := filepath.Base(archPath)
	uri := "https://s3." + region + ".cloud-object-storage.appdomain.cloud" + "/" + bucket + "/" + baseName

	f, err := os.Open(archPath)
	if err != nil {
		fmt.Println(err)
	}

	reader := bufio.NewReader(f)

	client := &http.Client{}
	r, err := http.NewRequest(http.MethodPut, uri, reader)
	if err != nil {
		fmt.Println(err)
	}
	r.Header.Set("Content-Type", "application/octet-stream")

	r.Header.Set("Authorization", "Bearer "+s.token)

	res, err := client.Do(r)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(string(body))

	return nil
}

// DeleteFromBucket deletes key from config's bucket
func (s *Objects) DeleteFromBucket(config *types.Config, key string) error {
	return nil
}
