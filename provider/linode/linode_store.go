package linode

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/nanovms/ops/types"
)

// Objects provides Linode Object Storage related operations
type Objects struct{}

// URLResponse provides a url for uploading an image to Linode.
type URLResponse struct {
	Link string `json:"upload_to"`
}

func getURL(imgName string) string {
	token := os.Getenv("TOKEN")
	client := &http.Client{}

	s := `{
		"label": "` + imgName + `",
		"region": "us-west"
	}`

	reqBody := []byte(s)

	uri := "https://api.linode.com/v4/images/upload"
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	ur := &URLResponse{}
	err = json.Unmarshal(body, &ur)
	if err != nil {
		fmt.Println(err)
	}

	delval, err := url.QueryUnescape(ur.Link)
	if err != nil {
		fmt.Println(err)
	}

	return delval
}

func uploadImage(uri string, archPath string) {
	f, err := os.Open(archPath)
	if err != nil {
		fmt.Println(err)
	}

	reader := bufio.NewReader(f)
	content, err := io.ReadAll(reader)
	if err != nil {
		fmt.Println(err)
	}

	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write(content)
	w.Close()

	slen := strconv.Itoa(len(buf.Bytes()))

	req, err := http.NewRequest("PUT", uri, bytes.NewReader(buf.Bytes()))
	if err != nil {
		fmt.Println(err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Length", slen)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	_, err = io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

}

// CopyToBucket copies archive to bucket
func (s *Objects) CopyToBucket(config *types.Config, archPath string) error {
	fmt.Println("copying to bucket..")

	imgName := config.CloudConfig.ImageName

	link := getURL(imgName)

	uploadImage(link, archPath)

	return nil
}

// DeleteFromBucket deletes key from config's bucket
func (s *Objects) DeleteFromBucket(config *types.Config, key string) error {
	return nil
}
