package lepton

import (
	"context"
	"strings"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/dns/v1"
)

// FindOrCreateZoneIDByName searches for a DNS zone with the name passed by argument and if it doesn't exist it creates one
func (p *GCloud) FindOrCreateZoneIDByName(config *Config, dnsName string) (string, error) {
	zoneName := strings.Split(dnsName, ".")[0]
	zone, err := p.dnsService.ManagedZones.Get(config.CloudConfig.ProjectID, zoneName).Do()
	if err != nil || zone == nil {
		managedZone := &dns.ManagedZone{
			Name:        zoneName,
			Description: zoneName,
			DnsName:     dnsName + ".",
		}

		_, err = p.dnsService.ManagedZones.Create(config.CloudConfig.ProjectID, managedZone).Do()
		if err != nil {
			return "", err
		}
	}

	return zoneName, nil
}

// DeleteZoneRecordIfExists deletes a record from a DNS zone if it exists
func (p *GCloud) DeleteZoneRecordIfExists(config *Config, zoneID string, recordName string) error {
	recordsResponse, err := p.dnsService.ResourceRecordSets.List(config.CloudConfig.ProjectID, zoneID).Do()
	if err != nil {
		return err
	}

	for _, record := range recordsResponse.Rrsets {
		if record.Name == recordName && record.Type == "A" {
			_, err = p.dnsService.Changes.Create(config.CloudConfig.ProjectID, zoneID, &dns.Change{
				Deletions: []*dns.ResourceRecordSet{record},
			}).Do()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// CreateZoneRecord creates a record in a DNS zone
func (p *GCloud) CreateZoneRecord(config *Config, zoneID string, record *DNSRecord) error {
	resource := &dns.ResourceRecordSet{
		Name:    record.Name,
		Type:    record.Type,
		Rrdatas: []string{record.IP},
		Ttl:     int64(record.TTL),
	}

	_, err := p.dnsService.Changes.Create(config.CloudConfig.ProjectID, zoneID, &dns.Change{
		Additions: []*dns.ResourceRecordSet{resource},
	}).Do()
	if err != nil {
		return err
	}
	return nil
}

func (p *GCloud) getDNSService() (*dns.Service, error) {
	context := context.TODO()
	_, err := google.FindDefaultCredentials(context)
	if err != nil {
		return nil, err
	}

	dnsService, err := dns.NewService(context)
	if err != nil {
		return nil, err
	}

	return dnsService, nil
}
