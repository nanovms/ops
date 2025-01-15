//go:build aws || !onlyprovider

package aws

import (
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	awsRoute53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// FindOrCreateZoneIDByName searches for a DNS zone with the name passed by argument and if it doesn't exist it creates one
func (p *AWS) FindOrCreateZoneIDByName(config *types.Config, dnsName string) (string, error) {
	var zoneID string
	hostedZones, err := p.dnsService.ListHostedZonesByName(p.execCtx, &route53.ListHostedZonesByNameInput{DNSName: &dnsName})
	if err == nil && hostedZones.HostedZones == nil {
		reference := strconv.Itoa(int(time.Now().Unix()))

		createHostedZoneInput := &route53.CreateHostedZoneInput{
			CallerReference: &reference,
			Name:            &dnsName,
		}

		hostedZone, err := p.dnsService.CreateHostedZone(p.execCtx, createHostedZoneInput)
		if err != nil {
			return "", err
		}

		zoneID = *hostedZone.HostedZone.Id
	} else if err != nil {
		return "", err
	} else {
		zoneID = *hostedZones.HostedZones[0].Id
	}

	return zoneID, nil
}

// DeleteZoneRecordIfExists deletes a record from a DNS zone if it exists
func (p *AWS) DeleteZoneRecordIfExists(config *types.Config, zoneID string, recordName string) error {
	records, err := p.dnsService.ListResourceRecordSets(p.execCtx, &route53.ListResourceRecordSetsInput{HostedZoneId: &zoneID})
	if err != nil {
		return err
	}

	for _, record := range records.ResourceRecordSets {
		if *record.Name == recordName && record.Type == "A" {
			input := &route53.ChangeResourceRecordSetsInput{
				ChangeBatch: &awsRoute53Types.ChangeBatch{
					Changes: []awsRoute53Types.Change{
						{
							Action:            awsRoute53Types.ChangeActionDelete,
							ResourceRecordSet: &record,
						},
					},
				},
				HostedZoneId: aws.String(zoneID),
			}

			_, err = p.dnsService.ChangeResourceRecordSets(p.execCtx, input)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// CreateZoneRecord creates a record in a DNS zone
func (p *AWS) CreateZoneRecord(config *types.Config, zoneID string, record *lepton.DNSRecord) error {
	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &awsRoute53Types.ChangeBatch{
			Changes: []awsRoute53Types.Change{
				{
					Action: awsRoute53Types.ChangeActionCreate,
					ResourceRecordSet: &awsRoute53Types.ResourceRecordSet{
						Name: aws.String(record.Name),
						ResourceRecords: []awsRoute53Types.ResourceRecord{
							{
								Value: aws.String(record.IP),
							},
						},
						TTL:  aws.Int64(int64(record.TTL)),
						Type: awsRoute53Types.RRType(strings.ToUpper(record.Type)),
					},
				},
			},
		},
		HostedZoneId: aws.String(zoneID),
	}

	_, err := p.dnsService.ChangeResourceRecordSets(p.execCtx, input)
	if err != nil {
		return err
	}

	return nil
}
