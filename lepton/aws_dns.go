package lepton

import (
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

// FindOrCreateZoneIDByName searches for a DNS zone with the name passed by argument and if it doesn't exist it creates one
func (p *AWS) FindOrCreateZoneIDByName(config *Config, dnsName string) (string, error) {
	dnsService, err := p.getDNSService(config)
	if err != nil {
		return "", err
	}

	var zoneID string
	hostedZones, err := dnsService.ListHostedZonesByName(&route53.ListHostedZonesByNameInput{DNSName: &dnsName})
	if err == nil && hostedZones.HostedZones == nil {
		reference := strconv.Itoa(int(time.Now().Unix()))

		createHostedZoneInput := &route53.CreateHostedZoneInput{
			CallerReference: &reference,
			Name:            &dnsName,
		}

		hostedZone, err := dnsService.CreateHostedZone(createHostedZoneInput)
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
func (p *AWS) DeleteZoneRecordIfExists(config *Config, zoneID string, recordName string) error {
	dnsService, err := p.getDNSService(config)
	if err != nil {
		return err
	}

	records, err := dnsService.ListResourceRecordSets(&route53.ListResourceRecordSetsInput{HostedZoneId: &zoneID})
	if err != nil {
		return err
	}

	for _, record := range records.ResourceRecordSets {
		if *record.Name == recordName && *record.Type == "A" {
			input := &route53.ChangeResourceRecordSetsInput{
				ChangeBatch: &route53.ChangeBatch{
					Changes: []*route53.Change{
						{
							Action:            aws.String("DELETE"),
							ResourceRecordSet: record,
						},
					},
				},
				HostedZoneId: aws.String(zoneID),
			}

			_, err = dnsService.ChangeResourceRecordSets(input)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// CreateZoneRecord creates a record in a DNS zone
func (p *AWS) CreateZoneRecord(config *Config, zoneID string, record *DNSRecord) error {
	dnsService, err := p.getDNSService(config)
	if err != nil {
		return err
	}

	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("CREATE"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(record.Name),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(record.IP),
							},
						},
						TTL:  aws.Int64(int64(record.TTL)),
						Type: aws.String(record.Type),
					},
				},
			},
		},
		HostedZoneId: aws.String(zoneID),
	}

	_, err = dnsService.ChangeResourceRecordSets(input)
	if err != nil {
		return err
	}

	return nil
}
