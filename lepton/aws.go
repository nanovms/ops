package lepton

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ebs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/route53"
)

// AWS contains all operations for AWS
type AWS struct {
	Storage       *S3
	dnsService    *route53.Route53
	volumeService *ebs.EBS
	session       *session.Session
	ec2           *ec2.EC2
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
func (p *AWS) Initialize(config *ProviderConfig) error {
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
			Region: aws.String(config.Zone),
		},
	)
	if err != nil {
		return err
	}
	// initialize aws services
	p.session = session
	p.dnsService = route53.New(session)
	p.ec2 = ec2.New(session)
	p.volumeService = ebs.New(session)

	_, err = p.ec2.DescribeRegions(&ec2.DescribeRegionsInput{RegionNames: aws.StringSlice([]string{config.Zone})})
	if err != nil {
		return fmt.Errorf("region with name %v is invalid", config.Zone)
	}

	return nil
}

// buildAwsTags converts configuration tags to AWS tags and returns the resource name. The defaultName is overridden if there is a tag with key name
func buildAwsTags(configTags []Tag, defaultName string) ([]*ec2.Tag, string) {
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

// GetInstanceByID returns the instance with the id passed by argument if it exists
func (p *AWS) GetInstanceByID(ctx *Context, id string) (*CloudInstance, error) {
	var filters []*ec2.Filter

	filters = append(filters, &ec2.Filter{Name: aws.String("tag:Name"), Values: aws.StringSlice([]string{id})})

	instances := getAWSInstances(ctx.config.CloudConfig.Zone, filters)

	if len(instances) == 0 {
		return nil, ErrInstanceNotFound(id)
	}

	return &instances[0], nil
}

// GetStorage returns storage interface for cloud provider
func (p *AWS) GetStorage() Storage {
	return p.Storage
}
