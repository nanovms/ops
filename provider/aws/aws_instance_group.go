//go:build aws || !onlyprovider

package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	awsAutoscalingTypes "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	awsEc2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/nanovms/ops/log"
)

// LaunchTemplateInput The parameters for a launch template.
type LaunchTemplateInput struct {
	AutoScalingGroup         string
	ImageID                  string
	InstanceNetworkInterface *awsEc2Types.InstanceNetworkInterfaceSpecification
	InstanceProfileName      string
	InstanceType             string
	LaunchTemplateName       string
	Tags                     []awsEc2Types.Tag
}

// launchTemplateInstanceNetworkInterfaceSpecificationRequest
// convert from InstanceNetworkInterfaceSpecification to LaunchTemplateInstanceNetworkInterfaceSpecificationRequest
func (lti LaunchTemplateInput) launchTemplateInstanceNetworkInterfaceSpecificationRequest() *awsEc2Types.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest {
	if lti.InstanceNetworkInterface == nil {
		return nil
	}

	req := &awsEc2Types.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{
		DeleteOnTermination: lti.InstanceNetworkInterface.DeleteOnTermination,
		DeviceIndex:         lti.InstanceNetworkInterface.DeviceIndex,
		Groups:              lti.InstanceNetworkInterface.Groups,
		PrivateIpAddress:    lti.InstanceNetworkInterface.PrivateIpAddress,
		SubnetId:            lti.InstanceNetworkInterface.SubnetId,
	}

	for _, item := range lti.InstanceNetworkInterface.Ipv6Addresses {
		req.Ipv6Addresses = append(req.Ipv6Addresses, awsEc2Types.InstanceIpv6AddressRequest{
			Ipv6Address: item.Ipv6Address,
		})
	}

	return req
}

// autoScalingLaunchTemplate
// the steps create launchTemplate - modify launchTemplate to default version and update launchTemplate to auto scaling group
func (p *AWS) autoScalingLaunchTemplate(req *LaunchTemplateInput) error {
	result, err := p.createLaunchTemplate(req)
	if err != nil {
		return err
	}

	log.Info("Created launch template", *result.LaunchTemplate.LaunchTemplateName)

	if err := p.modifyLaunchTemplate(fmt.Sprintf("%d", *result.LaunchTemplate.LatestVersionNumber), req.LaunchTemplateName); err != nil {
		return err
	}

	return p.updateAutoScalingGroup(req.AutoScalingGroup, req.LaunchTemplateName)
}

// createLaunchTemplate
// build CreateLaunchTemplateInput struct from LaunchTemplateInput
// and call CreateLaunchTemplate API for Amazon Elastic Compute Cloud.
func (p *AWS) createLaunchTemplate(req *LaunchTemplateInput) (*ec2.CreateLaunchTemplateOutput, error) {
	input := &ec2.CreateLaunchTemplateInput{
		LaunchTemplateData: &awsEc2Types.RequestLaunchTemplateData{
			BlockDeviceMappings: []awsEc2Types.LaunchTemplateBlockDeviceMappingRequest{
				{
					DeviceName: aws.String("/dev/sda1"),
					Ebs: &awsEc2Types.LaunchTemplateEbsBlockDeviceRequest{
						DeleteOnTermination: aws.Bool(true),
						VolumeType:          awsEc2Types.VolumeTypeGp2,
					},
				},
			},
			ImageId:      aws.String(req.ImageID),
			InstanceType: awsEc2Types.InstanceType(req.InstanceType),
			Monitoring: &awsEc2Types.LaunchTemplatesMonitoringRequest{
				Enabled: aws.Bool(true),
			},
			NetworkInterfaces: []awsEc2Types.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{
				*req.launchTemplateInstanceNetworkInterfaceSpecificationRequest(),
			},
			TagSpecifications: []awsEc2Types.LaunchTemplateTagSpecificationRequest{
				{
					ResourceType: awsEc2Types.ResourceTypeInstance,
					Tags:         req.Tags,
				},
			},
		},
		LaunchTemplateName: aws.String(req.LaunchTemplateName),
		VersionDescription: aws.String(req.LaunchTemplateName),
	}

	if req.InstanceProfileName != "" {
		input.LaunchTemplateData.IamInstanceProfile = &awsEc2Types.LaunchTemplateIamInstanceProfileSpecificationRequest{
			Name: aws.String(req.InstanceProfileName),
		}
	}

	return p.ec2.CreateLaunchTemplate(p.execCtx, input)
}

// modifyLaunchTemplate from one version to DefaultVersion
func (p *AWS) modifyLaunchTemplate(version string, ltName string) error {
	params := &ec2.ModifyLaunchTemplateInput{
		DefaultVersion:     aws.String(version),
		LaunchTemplateName: aws.String(ltName),
	}
	_, err := p.ec2.ModifyLaunchTemplate(p.execCtx, params)
	return err
}

// updateAutoScalingGroup
// build UpdateAutoScalingGroupInput and Updates the configuration for the specified Auto Scaling group.
func (p *AWS) updateAutoScalingGroup(asgName string, ltName string) error {
	params := &autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(asgName),
		LaunchTemplate: &awsAutoscalingTypes.LaunchTemplateSpecification{
			LaunchTemplateName: aws.String(ltName),
			Version:            aws.String("$Default"),
		},
	}

	if _, err := p.asg.UpdateAutoScalingGroup(p.execCtx, params); err != nil {
		return err
	}

	return nil
}
