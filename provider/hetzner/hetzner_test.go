package hetzner

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
	"github.com/nanovms/ops/lepton"
)

func strPtr(v string) *string { return &v }

func timePtr(v time.Time) *time.Time { return &v }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(t *testing.T, status int, body interface{}) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}
	resp := &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(data)),
	}
	resp.Header.Set("Content-Type", "application/json")
	return resp
}

func TestHetznerGetImages(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/images" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		page := r.URL.Query().Get("page")
		if page == "" || page == "1" {
			created := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			return jsonResponse(t, http.StatusOK, schema.ImageListResponse{
				Images: []schema.Image{
					{
						ID:       1,
						Name:     strPtr("snapshot-1"),
						Status:   string(hcloud.ImageStatusAvailable),
						DiskSize: 1024,
						Created:  timePtr(created),
						Labels: map[string]string{
							opsLabelKey:          opsLabelValue,
							opsImageNameLabelKey: "app-image",
						},
					},
				},
			}), nil
		}
		return jsonResponse(t, http.StatusOK, schema.ImageListResponse{}), nil
	})

	client := hcloud.NewClient(
		hcloud.WithToken("test-token"),
		hcloud.WithHTTPClient(&http.Client{Transport: transport}),
		hcloud.WithEndpoint("http://example.com"),
	)

	p := &Hetzner{Client: client}

	images, err := p.GetImages(new(lepton.Context), "")
	if err != nil {
		t.Fatalf("GetImages returned error: %v", err)
	}

	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}

	img := images[0]
	if img.ID != "1" {
		t.Fatalf("expected image ID 1, got %s", img.ID)
	}
	if img.Name != "app-image" {
		t.Fatalf("expected image name app-image, got %s", img.Name)
	}
}

func TestHetznerGetInstances(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/servers" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		page := r.URL.Query().Get("page")
		if page == "" || page == "1" {
			created := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
			return jsonResponse(t, http.StatusOK, schema.ServerListResponse{
				Servers: []schema.Server{
					{
						ID:      10,
						Name:    "ops-instance",
						Status:  string(hcloud.ServerStatusRunning),
						Created: created,
						PublicNet: schema.ServerPublicNet{
							IPv4: schema.ServerPublicNetIPv4{IP: "192.0.2.1"},
							IPv6: schema.ServerPublicNetIPv6{IP: "2001:db8::1"},
						},
						PrivateNet: []schema.ServerPrivateNet{
							{IP: "10.0.0.5"},
						},
						Labels: map[string]string{
							opsLabelKey:          opsLabelValue,
							opsImageNameLabelKey: "app-image",
						},
					},
					{
						ID:      11,
						Name:    "external-instance",
						Status:  string(hcloud.ServerStatusRunning),
						Created: created,
						Labels:  map[string]string{},
					},
				},
			}), nil
		}
		return jsonResponse(t, http.StatusOK, schema.ServerListResponse{}), nil
	})

	client := hcloud.NewClient(
		hcloud.WithToken("test-token"),
		hcloud.WithHTTPClient(&http.Client{Transport: transport}),
		hcloud.WithEndpoint("http://example.com"),
	)

	p := &Hetzner{Client: client}

	instances, err := p.GetInstances(new(lepton.Context))
	if err != nil {
		t.Fatalf("GetInstances returned error: %v", err)
	}

	if len(instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(instances))
	}

	inst := instances[0]
	if inst.ID != "10" {
		t.Fatalf("expected ID 10, got %s", inst.ID)
	}
	if inst.Name != "ops-instance" {
		t.Fatalf("expected name ops-instance, got %s", inst.Name)
	}
	if inst.Image != "app-image" {
		t.Fatalf("expected image label app-image, got %s", inst.Image)
	}
	if len(inst.PrivateIps) != 1 || inst.PrivateIps[0] != "10.0.0.5" {
		t.Fatalf("unexpected private ips %+v", inst.PrivateIps)
	}
}

// Suppress unused warnings for helpers when tests are compiled without execution paths.
var _ = context.Background
