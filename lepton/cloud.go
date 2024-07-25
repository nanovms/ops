package lepton

import (
	"encoding/json"
	"strconv"
	"time"
)

// CloudImage abstracts images for various cloud providers
type CloudImage struct {
	ID      string
	Name    string
	Status  string
	Size    int64
	Path    string
	Created time.Time
	Tag     string // could merge w/below
	Labels  []string
}

// CloudInstance represents the instance that widely use in different
// Cloud Providers.
// mainly used for formatting standard response from any cloud provider
type CloudInstance struct {
	ID          string
	Name        string
	Status      string
	Created     string // TODO: prob. should be datetime w/helpers for human formatting
	PrivateIps  []string
	PublicIps   []string
	Ports       []string
	Image       string
	FreeMemory  int64
	TotalMemory int64
}

// HumanMem returns the used / total memory in human format.
func (c CloudInstance) HumanMem() string {
	return strconv.FormatInt((c.TotalMemory-c.FreeMemory), 10) + "mb / " + strconv.FormatInt(c.TotalMemory, 10) + "mb"
}

// MarshalJSON ensures correct json serialization of potential null
// vals.
func (c CloudInstance) MarshalJSON() ([]byte, error) {
	type Alias CloudInstance

	a := struct {
		Alias
	}{
		Alias: (Alias)(c),
	}

	if a.PublicIps == nil {
		a.PublicIps = make([]string, 0)
	}

	if a.PrivateIps == nil {
		a.PrivateIps = make([]string, 0)
	}

	if a.Ports == nil {
		a.Ports = make([]string, 0)
	}

	return json.Marshal(a)
}
