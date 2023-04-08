//go:build azure || !onlyprovider

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// FindOrCreateZoneIDByName searches for a DNS zone with the name passed by argument and if it doesn't exist it creates one
func (a *Azure) FindOrCreateZoneIDByName(config *types.Config, dnsName string) (string, error) {
	service := dns.NewZonesClient(a.subID)
	authr, _ := a.GetResourceManagementAuthorizer()
	service.Authorizer = authr

	ctx := context.TODO()
	zonesListResponse, err := service.List(ctx, nil)
	if err != nil {
		return "", err
	}

	zones := zonesListResponse.Values()
	if len(zones) == 0 {
		location := "global"
		zone := dns.Zone{
			Name:     &dnsName,
			Location: &location,
		}

		_, err := service.CreateOrUpdate(ctx, a.groupName, dnsName, zone, "", "")
		if err != nil {
			return "", err
		}
	}

	return dnsName, nil
}

// DeleteZoneRecordIfExists deletes a record from a DNS zone if it exists
func (a *Azure) DeleteZoneRecordIfExists(config *types.Config, zoneID string, recordName string) error {
	return nil
}

// CreateZoneRecord creates a record in a DNS zone
func (a *Azure) CreateZoneRecord(config *types.Config, zoneID string, record *lepton.DNSRecord) error {
	service := dns.NewRecordSetsClient(a.subID)
	authr, _ := a.GetResourceManagementAuthorizer()
	service.Authorizer = authr

	// remove trailing dot if it exists
	if record.Name[len(record.Name)-1] == '.' {
		record.Name = record.Name[:len(record.Name)-1]
	}

	ctx := context.TODO()
	dnsRecord := dns.RecordSet{
		Name: &record.Name,
		Type: &record.Type,
		RecordSetProperties: &dns.RecordSetProperties{
			ARecords: &[]dns.ARecord{
				{Ipv4Address: &record.IP},
			},
			TTL: to.Int64Ptr(int64(record.TTL)),
		},
	}

	_, err := service.CreateOrUpdate(ctx, a.groupName, zoneID, record.Name, dns.RecordType(record.Type), dnsRecord, "", "")
	if err != nil {
		return err
	}

	return nil
}
