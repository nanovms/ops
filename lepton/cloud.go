package lepton

import "time"

// CloudImage abstracts images for various cloud providers
type CloudImage struct {
	ID      string
	Name    string
	Status  string
	Size    int64
	Path    string
	Created time.Time
}

// CloudInstance represents the instance that widely use in different
// Cloud Providers.
type CloudInstance struct {
	ID         string
	Name       string
	Status     string
	Created    string // TODO: prob. should be datetime w/helpers for human formatting
	PrivateIps []string
	PublicIps  []string
	Ports      []string
	Image      string
}
