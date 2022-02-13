package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ebs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// AWS contains all operations for AWS
type AWS struct {
	Storage       *S3
	dnsService    *route53.Route53
	volumeService *ebs.EBS
	session       *session.Session
	ec2           *ec2.EC2
}

// strips any zone qualifier from 'zone' string
// some AWS API calls only want region even if full region-zone is used
// elsewhere in the same call
// FIXME
func stripZone(zone string) string {
	return strings.TrimRight(zone, "abc")
}

func loadAWSCreds() (err error) {
	foundValidCredentials := false

	fileCreds := credentials.NewSharedCredentials("", "")

	_, err = fileCreds.Get()
	if err == nil {
		foundValidCredentials = true
	}

	envCreds := credentials.NewEnvCredentials()

	_, err = envCreds.Get()
	if err == nil {
		foundValidCredentials = true
	}

	if foundValidCredentials {
		err = nil
	}

	return
}

// Initialize AWS related things
func (p *AWS) Initialize(config *types.ProviderConfig) error {
	p.Storage = &S3{}

	if config.Zone == "" {
		return fmt.Errorf("Zone missing")
	}

	err := loadAWSCreds()
	if err != nil {
		return err
	}

	session, err := session.NewSession(
		&aws.Config{
			Region: aws.String(stripZone(config.Zone)),
		},
	)
	if err != nil {
		return err
	}
	// initialize aws services
	p.session = session
	p.dnsService = route53.New(session)
	p.ec2 = ec2.New(session)
	p.volumeService = ebs.New(session,
		aws.NewConfig().
			WithRegion(stripZone(config.Zone)).
			WithMaxRetries(7))

	_, err = p.ec2.DescribeRegions(&ec2.DescribeRegionsInput{RegionNames: aws.StringSlice([]string{stripZone(config.Zone)})})
	if err != nil {
		return fmt.Errorf("region with name %v is invalid", config.Zone)
	}

	return nil
}

// buildAwsTags converts configuration tags to AWS tags and returns the resource name. The defaultName is overridden if there is a tag with key name
func buildAwsTags(configTags []types.Tag, defaultName string) ([]*ec2.Tag, string) {
	tags := []*ec2.Tag{}
	var nameSpecified bool
	name := defaultName

	for _, tag := range configTags {
		tags = append(tags, &ec2.Tag{Key: aws.String(tag.Key), Value: aws.String(tag.Value)})
		if tag.Key == "Name" {
			nameSpecified = true
			name = tag.Value
		}
	}

	if !nameSpecified {
		tags = append(tags, &ec2.Tag{
			Key:   aws.String("Name"),
			Value: aws.String(name),
		})
	}

	tags = append(tags, &ec2.Tag{
		Key:   aws.String("CreatedBy"),
		Value: aws.String("ops"),
	})

	return tags, name
}

func (p *AWS) getNameTag(tags []*ec2.Tag) *ec2.Tag {
	for _, tag := range tags {
		if *tag.Key == "Name" {
			return tag
		}
	}

	return nil
}

// GetStorage returns storage interface for cloud provider
func (p *AWS) GetStorage() lepton.Storage {
	return p.Storage
}
