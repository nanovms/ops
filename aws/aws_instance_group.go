package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/nanovms/ops/log"
)

// LaunchTemplateInput The parameters for a launch template.
type LaunchTemplateInput struct {
	AutoScalingGroup         string
	ImageID                  string
	InstanceNetworkInterface *ec2.InstanceNetworkInterfaceSpecification
	InstanceProfileName      string
	InstanceType             string
	LaunchTemplateName       string
	Tags                     []*ec2.Tag
}

// launchTemplateInstanceNetworkInterfaceSpecificationRequest
// convert from InstanceNetworkInterfaceSpecification to LaunchTemplateInstanceNetworkInterfaceSpecificationRequest
func (lti LaunchTemplateInput) launchTemplateInstanceNetworkInterfaceSpecificationRequest() *ec2.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest {
	if lti.InstanceNetworkInterface == nil {
		return nil
	}

	req := &ec2.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{
		DeleteOnTermination: lti.InstanceNetworkInterface.DeleteOnTermination,
		DeviceIndex:         lti.InstanceNetworkInterface.DeviceIndex,
		Groups:              lti.InstanceNetworkInterface.Groups,
		PrivateIpAddress:    lti.InstanceNetworkInterface.PrivateIpAddress,
		SubnetId:            lti.InstanceNetworkInterface.SubnetId,
	}

	for _, item := range lti.InstanceNetworkInterface.Ipv6Addresses {
		req.Ipv6Addresses = append(req.Ipv6Addresses, &ec2.InstanceIpv6AddressRequest{
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
		LaunchTemplateData: &ec2.RequestLaunchTemplateData{
			BlockDeviceMappings: []*ec2.LaunchTemplateBlockDeviceMappingRequest{
				{
					DeviceName: aws.String("/dev/sda1"),
					Ebs: &ec2.LaunchTemplateEbsBlockDeviceRequest{
						DeleteOnTermination: aws.Bool(true),
						VolumeType:          aws.String("gp2"),
					},
				},
			},
			ImageId:      aws.String(req.ImageID),
			InstanceType: aws.String(req.InstanceType),
			Monitoring: &ec2.LaunchTemplatesMonitoringRequest{
				Enabled: aws.Bool(true),
			},
			NetworkInterfaces: []*ec2.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{
				req.launchTemplateInstanceNetworkInterfaceSpecificationRequest(),
			},
			TagSpecifications: []*ec2.LaunchTemplateTagSpecificationRequest{
				{
					ResourceType: aws.String("instance"),
					Tags:         req.Tags,
				},
			},
		},
		LaunchTemplateName: aws.String(req.LaunchTemplateName),
		VersionDescription: aws.String(req.LaunchTemplateName),
	}

	if req.InstanceProfileName != "" {
		input.LaunchTemplateData.IamInstanceProfile = &ec2.LaunchTemplateIamInstanceProfileSpecificationRequest{
			Name: aws.String(req.InstanceProfileName),
		}
	}

	return p.ec2.CreateLaunchTemplate(input)
}

// modifyLaunchTemplate from one version to DefaultVersion
func (p *AWS) modifyLaunchTemplate(version string, ltName string) error {
	params := &ec2.ModifyLaunchTemplateInput{
		DefaultVersion:     aws.String(version),
		LaunchTemplateName: aws.String(ltName),
	}
	_, err := p.ec2.ModifyLaunchTemplate(params)
	return err
}

// updateAutoScalingGroup
// build UpdateAutoScalingGroupInput and Updates the configuration for the specified Auto Scaling group.
func (p *AWS) updateAutoScalingGroup(asgName string, ltName string) error {
	asgCli := autoscaling.New(p.session)

	params := &autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(asgName),
		LaunchTemplate: &autoscaling.LaunchTemplateSpecification{
			LaunchTemplateName: aws.String(ltName),
			Version:            aws.String("$Default"),
		},
	}

	if _, err := asgCli.UpdateAutoScalingGroup(params); err != nil {
		return err
	}

	return nil
}
