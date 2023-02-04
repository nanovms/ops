package digitalocean

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/nanovms/ops/lepton"
)

var (
	mux *http.ServeMux

	ctx = context.TODO()

	client *godo.Client

	server *httptest.Server
)

func TestDoGetImages(t *testing.T) {
	setup()
	defer teardown()
	mux.HandleFunc("/v2/images", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"images": [
				{
					"id": 1,
                    "name": "test1",
                    "status": "test",
                    "created_at": "2020-09-04T06:50:46Z"
				},
				{
					"id": 2,
                    "name": "test2",
                    "status": "test",
                    "created_at": "2020-09-04T06:50:46Z"
				}
			],
			"meta": {
				"total": 2
			}
		}`)
	})
	do := &DigitalOcean{
		Client: client,
	}
	images, err := do.GetImages(&lepton.Context{})
	if err != nil {
		t.Error(err)
	}

	item1CreatedAt, _ := time.Parse("2006-01-02T15:04:05Z", "2020-09-04T06:50:46Z")
	item2CreatedAt, _ := time.Parse("2006-01-02T15:04:05Z", "2020-09-04T06:50:46Z")

	expectedResult := []lepton.CloudImage{
		{ID: "1", Name: "test1", Status: "test", Created: item1CreatedAt},
		{ID: "2", Name: "test2", Status: "test", Created: item2CreatedAt},
	}
	if !reflect.DeepEqual(images, expectedResult) {
		t.Fail()
	}
}

func setup() {
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	client = godo.NewClient(nil)
	url, _ := url.Parse(server.URL)
	client.BaseURL = url
}

func teardown() {
	server.Close()
}
