package lepton

import (
	"os"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/dns/v2/recordsets"
	"github.com/gophercloud/gophercloud/openstack/dns/v2/zones"
	"github.com/nanovms/ops/config"
)

// FindOrCreateZoneIDByName searches for a DNS zone with the name passed by argument and if it doesn't exist it creates one
func (o *OpenStack) FindOrCreateZoneIDByName(config *config.Config, dnsName string) (string, error) {
	dnsClient, err := o.getDNSClient()
	if err != nil {
		return "", err
	}

	var zoneID string

	opts := zones.ListOpts{
		Name: dnsName + ".",
	}
	allPages, err := zones.List(dnsClient, opts).AllPages()
	if err != nil {
		return "", err
	}

	allZones, err := zones.ExtractZones(allPages)
	if err != nil {
		return "", err
	}

	if len(allZones) == 0 {
		createOpts := zones.CreateOpts{
			Name:  dnsName + ".",
			Email: "admin@" + dnsName,
		}
		zone, err := zones.Create(dnsClient, createOpts).Extract()
		if err != nil {
			return "", err
		}

		zoneID = zone.ID
	} else {
		zoneID = allZones[0].ID
	}

	return zoneID, nil
}

// DeleteZoneRecordIfExists deletes a record from a DNS zone if it exists
func (o *OpenStack) DeleteZoneRecordIfExists(config *config.Config, zoneID string, recordName string) error {
	dnsClient, err := o.getDNSClient()
	if err != nil {
		return err
	}

	listOpts := recordsets.ListOpts{
		Type: "A",
		Name: recordName,
	}
	allPages, err := recordsets.ListByZone(dnsClient, zoneID, listOpts).AllPages()

	allRecords, err := recordsets.ExtractRecordSets(allPages)
	if err != nil {
		return err
	}

	for _, record := range allRecords {
		err := recordsets.Delete(dnsClient, zoneID, record.ID).ExtractErr()
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateZoneRecord creates a record in a DNS zone
func (o *OpenStack) CreateZoneRecord(config *config.Config, zoneID string, record *DNSRecord) error {
	dnsClient, err := o.getDNSClient()
	if err != nil {
		return err
	}

	createOpts := recordsets.CreateOpts{
		Name:    record.Name,
		Records: []string{record.IP},
		TTL:     record.TTL,
		Type:    record.Type,
	}

	_, err = recordsets.Create(dnsClient, zoneID, createOpts).Extract()
	if err != nil {
		return err
	}

	return nil
}

func (o *OpenStack) getDNSClient() (*gophercloud.ServiceClient, error) {
	return openstack.NewDNSV2(o.provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
}
