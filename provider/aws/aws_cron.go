//go:build aws || !onlyprovider

package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awsEc2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/olekukonko/tablewriter"
)

// Cron is a generic representation for a cloud scheduler such as an eventbridge schedule.
type Cron struct {
	ID        string
	Name      string
	State     string
	CreatedAt time.Time
}

// CreateCron creates an eventbridge schedule on AWS.
// a *lot* of this shares w/instance create and we should have it
// share..
func (p *AWS) CreateCron(ctx *lepton.Context, name string, schedule string) error {
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

	svc := p.ec2

	cloudConfig := ctx.Config().CloudConfig

	if cloudConfig.Flavor == "" {
		cloudConfig.Flavor = "t2.micro"
	}

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

	var sg *awsEc2Types.SecurityGroup

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

	scheduleName := name
	targetArn := "arn:aws:scheduler:::aws-sdk:ec2:runInstances"
	executionRoleArn := os.Getenv("EXECUTIONARN")

	scheduleExpression := schedule

	inputJSON := `{
	      "ImageId":"` + ami + `", 	
      "InstanceType":"` + cloudConfig.Flavor + `",
	      "MinCount":1,
	      "MaxCount":1,
	      "NetworkInterfaces":[
	          {
	          "DeleteOnTermination": true,
	          "DeviceIndex":         0,
			 "Groups": [
					"` + *sg.GroupId + `"
				],
	          "SubnetId":"` + *subnet.SubnetId + `"
	          }
	      ]
	  }`

	// you can add tags to a schedule group but not an indvidual
	// schedule - perhaps consider adding this at some point in the
	// future.
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

	fmt.Printf("Successfully created schedule %s to invoke %s every %s.\n", scheduleName, targetArn, schedule)
	return nil
}

// DeleteCron deletes an eventbridge schedule.
func (p *AWS) DeleteCron(ctx *lepton.Context, schedule string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}

	svc := scheduler.NewFromConfig(cfg)

	input := &scheduler.DeleteScheduleInput{
		Name: &schedule,
	}

	_, err = svc.DeleteSchedule(context.TODO(), input)
	if err != nil {
		if _, ok := err.(*types.ResourceNotFoundException); ok {
			fmt.Printf("Schedule '%s' not found.\n", schedule)
		} else {
			log.Fatalf("failed to delete schedule: %v", err)
		}
	} else {
		fmt.Printf("Schedule '%s' deleted successfully.\n", schedule)
	}

	return fmt.Errorf("operation not supported")
}

// EnableCron enables an eventbridge schedule.
// seems like these two want all the options simply to toggle it
// on/off..
func (p *AWS) EnableCron(ctx *lepton.Context, schedule string) error {
	cron, err := p.getCronByName(ctx, schedule)
	if err != nil {
		return err
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}

	svc := scheduler.NewFromConfig(cfg)

	_, err = svc.UpdateSchedule(context.TODO(), &scheduler.UpdateScheduleInput{
		Name:               &schedule,
		ScheduleExpression: cron.ScheduleExpression,
		Target:             cron.Target,
		FlexibleTimeWindow: cron.FlexibleTimeWindow,
		State:              types.ScheduleStateEnabled,
	})
	if err != nil {
		return err
	}

	return nil
}

// DisableCron disables an eventbridge schedule.
func (p *AWS) DisableCron(ctx *lepton.Context, schedule string) error {
	cron, err := p.getCronByName(ctx, schedule)
	if err != nil {
		return err
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}

	svc := scheduler.NewFromConfig(cfg)

	_, err = svc.UpdateSchedule(context.TODO(), &scheduler.UpdateScheduleInput{
		Name:               &schedule,
		ScheduleExpression: cron.ScheduleExpression,
		Target:             cron.Target,
		FlexibleTimeWindow: cron.FlexibleTimeWindow,
		State:              types.ScheduleStateDisabled,
	})
	if err != nil {
		return err
	}

	return nil
}

// ListCrons lists eventbridge schedules.
func (p *AWS) ListCrons(ctx *lepton.Context) error {
	crons, err := p.getCrons(ctx)
	if err != nil {
		return err
	}

	if ctx.Config().RunConfig.JSON {
		if err := json.NewEncoder(os.Stdout).Encode(crons); err != nil {
			return err
		}
	} else {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "Name", "State", "Created"})
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
		table.SetRowLine(true)

		for _, image := range crons {
			var row []string

			row = append(row, image.ID)
			row = append(row, image.Name)
			row = append(row, image.State)
			row = append(row, lepton.Time2Human(image.CreatedAt))

			table.Append(row)
		}

		table.Render()
	}
	return nil
}

func (p *AWS) getCronByName(ctx *lepton.Context, name string) (*scheduler.GetScheduleOutput, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return &scheduler.GetScheduleOutput{}, err
	}

	svc := scheduler.NewFromConfig(cfg)

	input := &scheduler.GetScheduleInput{
		Name: aws.String(name),
	}

	return svc.GetSchedule(context.TODO(), input)
}

func (p *AWS) getCrons(ctx *lepton.Context) ([]Cron, error) {
	var crons []Cron

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return crons, err
	}

	svc := scheduler.NewFromConfig(cfg)

	input := &scheduler.ListSchedulesInput{}

	result, err := svc.ListSchedules(context.TODO(), input)
	if err != nil {
		return crons, err
	}

	if len(result.Schedules) == 0 {
		fmt.Println("No schedules found.")
	} else {
		for _, schedule := range result.Schedules {

			c := Cron{
				ID:        *schedule.Arn,
				Name:      *schedule.Name,
				State:     string(schedule.State),
				CreatedAt: *schedule.CreationDate,
			}
			crons = append(crons, c)

			for result.NextToken != nil {
				input.NextToken = result.NextToken
				result, err = svc.ListSchedules(context.TODO(), input)
				if err != nil {
					return crons, err
				}
				for _, schedule := range result.Schedules {
					fmt.Printf("%+v\n", schedule)

					c := Cron{
						ID:        *schedule.Arn,
						Name:      *schedule.Name,
						State:     string(schedule.State),
						CreatedAt: *schedule.CreationDate,
					}
					crons = append(crons, c)

				}
			}
		}

	}

	return crons, err
}
