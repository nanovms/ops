package lepton

import (
	"context"
	"fmt"
	"github.com/digitalocean/godo"
	"net/http"
	"net/http/httptest"
	"net/url"
)

var (
	mux *http.ServeMux

	ctx = context.TODO()

	client *godo.Client

	server *httptest.Server
)

// Tests for ListImages using Golang examples.
// This tests the printed output to the console.
// More info at: https://golang.org/pkg/testing/#hdr-Examples
func Example_do_ListImages() {
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
	err := do.ListImages(&Context{})
	if err != nil {
		println(err.Error())
	}
	// Output:
	// +-------+--------+----------------------+
	// | NAME  | STATUS |       CREATED        |
	// +-------+--------+----------------------+
	// | test1 | test   | 2020-09-04T06:50:46Z |
	// +-------+--------+----------------------+
	// | test2 | test   | 2020-09-04T06:50:46Z |
	// +-------+--------+----------------------+
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
