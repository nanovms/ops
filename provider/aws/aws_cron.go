//go:build aws || !onlyprovider

package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awsEc2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
)

// a *lot* of this shares w/instance create and we should have it
// share..
func (p *AWS) CreateCron(ctx *lepton.Context, schedule string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("failed to load SDK config, %v", err)
	}

	ctx.Logger().Debug("getting aws images")
	result, err := getAWSImages(p.execCtx, p.ec2)
	if err != nil {
		ctx.Logger().Errorf("failed getting images")
		return err
	}

	var image *awsEc2Types.Image

	imgName := ctx.Config().CloudConfig.ImageName

	for i := 0; i < len(result.Images); i++ {
		if result.Images[i].Tags != nil {
			for _, tag := range result.Images[i].Tags {
				if *tag.Key == "Name" && *tag.Value == imgName {
					image = &result.Images[i]
					break
				}
			}
		}
	}

	ami := ""

	if image == nil {
		return fmt.Errorf("can't find ami with name %s", imgName)
	}

	ami = aws.ToString(image.ImageId)

	fmt.Println("using %s", ami)

	svc := p.ec2

	cloudConfig := ctx.Config().CloudConfig

	ctx.Logger().Debug("getting vpc")
	vpc, err := p.GetVPC(p.execCtx, ctx, svc)
	if err != nil {
		return err
	}

	if vpc == nil {
		ctx.Logger().Debugf("creating vpc with name %s", cloudConfig.VPC)
		vpc, err = p.CreateVPC(p.execCtx, ctx, svc)
		if err != nil {
			return err
		}
	}

	/*	var sg *awsEc2Types.SecurityGroup

		if cloudConfig.SecurityGroup != "" && cloudConfig.VPC != "" {
			ctx.Logger().Debugf("getting security group with name %s", cloudConfig.SecurityGroup)
			sg, err = p.GetSecurityGroup(p.execCtx, ctx, svc, vpc)
			if err != nil {
				return err
			}
		} else {
			iname := ctx.Config().RunConfig.InstanceName
			ctx.Logger().Debugf("creating new security group in vpc %s", *vpc.VpcId)
			sg, err = p.CreateSG(p.execCtx, ctx, svc, iname, *vpc.VpcId)
			if err != nil {
				return err
			}
		}
	*/

	ctx.Logger().Debug("getting subnet")
	var subnet *awsEc2Types.Subnet
	subnet, err = p.GetSubnet(p.execCtx, ctx, svc, *vpc.VpcId)
	if err != nil {
		return err
	}

	if subnet == nil {
		subnet, err = p.CreateSubnet(p.execCtx, ctx, vpc)
		if err != nil {
			return err
		}
	}

	client := scheduler.NewFromConfig(cfg)

	scheduleName := "itesting"
	targetArn := "arn:aws:scheduler:::aws-sdk:ec2:runInstances"
	executionRoleArn := os.Getenv("EXECUTIONARN")

	scheduleExpression := "rate(1 minutes)"

	inputJSON := `{
	      "ImageId":"` + ami + `", 	
      "InstanceType":"t2.micro",
	      "MinCount":1,
	      "MaxCount":1,
	      "NetworkInterfaces":[
	          {
	          "DeleteOnTermination": true,
	          "DeviceIndex":         0,
	          "SubnetId":"` + *subnet.SubnetId + `"
	          }
	      ]
	  }`

	input := &scheduler.CreateScheduleInput{
		Name:               &scheduleName,
		ScheduleExpression: &scheduleExpression,
		FlexibleTimeWindow: &types.FlexibleTimeWindow{Mode: types.FlexibleTimeWindowModeOff},
		Target: &types.Target{
			Arn:     &targetArn,
			RoleArn: &executionRoleArn,
			Input:   aws.String(inputJSON),
		},
	}

	_, err = client.CreateSchedule(context.TODO(), input)
	if err != nil {
		log.Fatalf("failed to create schedule, %v", err)
	}

	fmt.Printf("Successfully created schedule %s to invoke Lambda function %s every 5 minutes.\n", scheduleName, targetArn)
	return nil
}

func (p *AWS) DeleteCron(ctx *lepton.Context) error {
	return fmt.Errorf("operation not supported")
}

func (p *AWS) EnableCron(ctx *lepton.Context) error {
	return fmt.Errorf("operation not supported")
}

func (p *AWS) DisableCron(ctx *lepton.Context) error {
	return fmt.Errorf("operation not supported")
}

func (p *AWS) ListCrons(ctx *lepton.Context) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}

	svc := scheduler.NewFromConfig(cfg)

	input := &scheduler.ListSchedulesInput{
		// perhaps this should be 'nanos' or tagged?
		// GroupName
	}

	result, err := svc.ListSchedules(context.TODO(), input)
	if err != nil {
		log.Fatalf("failed to list schedules, %v", err)
	}

	if len(result.Schedules) == 0 {
		fmt.Println("No schedules found.")
	} else {
		fmt.Println("Found schedules:")
		for _, schedule := range result.Schedules {
			fmt.Printf("%+v\n", schedule)
			fmt.Printf("%+v\n", schedule.Arn)
		}

		for result.NextToken != nil {
			input.NextToken = result.NextToken
			result, err = svc.ListSchedules(context.TODO(), input)
			if err != nil {
				log.Fatalf("failed to list schedules (pagination), %v", err)
			}
			for _, schedule := range result.Schedules {
				fmt.Printf("%+v\n", schedule)
			}
		}
	}

	return fmt.Errorf("operation not supported")
}

/*
	if cloudConfig.Flavor == "" {
		cloudConfig.Flavor = "t2.micro"
	}

	// Create tags to assign to the instance
	tags, tagInstanceName := buildAwsTags(cloudConfig.Tags, ctx.Config().RunConfig.InstanceName)
	tags = append(tags, awsEc2Types.Tag{Key: aws.String("image"), Value: &imgName})

	instanceNIS := &awsEc2Types.InstanceNetworkInterfaceSpecification{
		DeleteOnTermination: aws.Bool(true),
		DeviceIndex:         aws.Int32(0),
		Groups: []string{
			aws.ToString(sg.GroupId),
		},
		SubnetId: aws.String(*subnet.SubnetId),
	}
*/
