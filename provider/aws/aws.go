//go:build aws || !onlyprovider

package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/ebs"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	awsEc2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// ProviderName of the cloud platform provider
const ProviderName = "aws"

// AWS Provider to interact with AWS cloud infrastructure
type AWS struct {
	execCtx       context.Context
	Storage       *S3
	dnsService    *route53.Client
	volumeService *ebs.Client
	ec2           *ec2.Client
	asg           *autoscaling.Client
	iam           *iam.Client
}

// NewProvider AWS
func NewProvider() *AWS {
	execCtx := context.Background()
	return &AWS{execCtx: execCtx}
}

// strips any zone qualifier from 'zone' string
// some AWS API calls only want region even if full region-zone is used
// elsewhere in the same call
// FIXME
func stripZone(zone string) string {
	return strings.TrimRight(zone, "abc")
}

// Initialize AWS related things
func (p *AWS) Initialize(config *types.ProviderConfig) error {
	p.Storage = &S3{}

	awsSdkConfig, err := GetAwsSdkConfig(p.execCtx, &config.Zone)
	if err != nil {
		return err
	}
	p.dnsService = route53.NewFromConfig(*awsSdkConfig)
	p.ec2 = ec2.NewFromConfig(*awsSdkConfig)
	p.volumeService = ebs.NewFromConfig(*awsSdkConfig)
	p.asg = autoscaling.NewFromConfig(*awsSdkConfig)
	p.iam = iam.NewFromConfig(*awsSdkConfig)

	_, err = p.ec2.DescribeRegions(p.execCtx, &ec2.DescribeRegionsInput{RegionNames: []string{stripZone(config.Zone)}})
	if err != nil {
		return fmt.Errorf("region with name %v is invalid", config.Zone)
	}

	return nil
}

// buildAwsTags converts configuration tags to AWS tags and returns the resource name. The defaultName is overridden if there is a tag with key name
func buildAwsTags(configTags []types.Tag, defaultName string) ([]awsEc2Types.Tag, string) {
	tags := []awsEc2Types.Tag{}
	var nameSpecified bool
	name := defaultName

	for _, tag := range configTags {
		tags = append(tags, awsEc2Types.Tag{Key: aws.String(tag.Key), Value: aws.String(tag.Value)})
		if tag.Key == "Name" {
			nameSpecified = true
			name = tag.Value
		}
	}

	if !nameSpecified {
		tags = append(tags, awsEc2Types.Tag{
			Key:   aws.String("Name"),
			Value: aws.String(name),
		})
	}

	tags = append(tags, awsEc2Types.Tag{
		Key:   aws.String("CreatedBy"),
		Value: aws.String("ops"),
	})

	return tags, name
}

func (p *AWS) getNameTag(tags []awsEc2Types.Tag) *awsEc2Types.Tag {
	for _, tag := range tags {
		if *tag.Key == "Name" {
			return &tag
		}
	}

	return nil
}

// GetStorage returns storage interface for cloud provider
func (p *AWS) GetStorage() lepton.Storage {
	return p.Storage
}
