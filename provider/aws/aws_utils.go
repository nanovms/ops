//go:build aws || !onlyprovider

package aws

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	awsEc2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	smithyTime "github.com/aws/smithy-go/time"
	smithyWaiter "github.com/aws/smithy-go/waiter"
)

// GetAwsSdkConfig creates and return an aws-sdk-v2 configuration
// object.
func GetAwsSdkConfig(execCtx context.Context, zone *string) (*aws.Config, error) {
	awsSdkConfigOpts := []func(*awsConfig.LoadOptions) error{}
	awsProfile := os.Getenv("AWS_PROFILE")
	if awsProfile != "" {
		awsSdkConfigOpts = append(awsSdkConfigOpts, awsConfig.WithSharedConfigProfile(awsProfile))
	}
	if zone != nil {
		awsSdkConfigOpts = append(awsSdkConfigOpts, awsConfig.WithRegion(stripZone(*zone)))
	}
	awsSdkConfig, err := awsConfig.LoadDefaultConfig(execCtx, awsSdkConfigOpts...)
	if err != nil {
		fmt.Println("Couldn't load default configuration. Have you set up your AWS account?")
		fmt.Println(err)
		return nil, err
	}
	return &awsSdkConfig, nil
}

// WaitUntilEc2InstanceTerminated waits until an EC2 instance is terminated. This is a blocking operation on the provided execution context.
// Ensure proper filters since this function expects a single EC2 given
// the input.
// Derived from NewSnapshotCompletedWaiter impl. at https://github.com/aws/aws-sdk-go-v2/blob/v1.32.8/service/ec2/api_op_DescribeSnapshots.go#L348
func WaitUntilEc2InstanceTerminated(execCtx context.Context, ec2Client *ec2.Client, describeInstancesInput *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	minDelay := 15 * time.Second
	maxDelay := minDelay + 1
	waitDur := 120 * maxDelay
	ctx, cancelFn := context.WithTimeout(execCtx, waitDur)
	defer cancelFn()
	remainingTime := waitDur

	var attempt int64
	for {
		attempt++
		start := time.Now()

		out, err := ec2Client.DescribeInstances(ctx, describeInstancesInput)

		if err != nil {
			return nil, err
		} else if len(out.Reservations) < 1 {
			return nil, fmt.Errorf("error while waiting for an EC2 to terminate. No EC2 were found for this input, %v", describeInstancesInput)
		} else if len(out.Reservations) != 1 {
			return nil, fmt.Errorf("this function expects one reservation so please adjust your filter, %v", describeInstancesInput)
		} else if len(out.Reservations[0].Instances) != 1 {
			return nil, fmt.Errorf("this function expects one instance so please adjust your filter, %v", describeInstancesInput)
		}

		instance := out.Reservations[0].Instances[0]

		if instance.State.Name == awsEc2Types.InstanceStateNameTerminated {
			return out, nil
		}

		remainingTime -= time.Since(start)
		if remainingTime < minDelay || remainingTime <= 0 {
			break
		}

		// compute exponential backoff between waiter retries
		delay, err := smithyWaiter.ComputeDelay(
			attempt, minDelay, maxDelay+1, remainingTime,
		)
		if err != nil {
			return nil, fmt.Errorf("error computing a waiter delay, %w", err)
		}

		remainingTime -= delay
		// sleep for the delay amount before invoking a request
		if err := smithyTime.SleepWithContext(ctx, delay); err != nil {
			return nil, fmt.Errorf("request cancelled while waiting, %w", err)
		}
	}
	return nil, fmt.Errorf("exceeded max wait time for an EC2 instance to terminate")
}

// WaitUntilEc2SnapshotCompleted waits for snapshot completion. This is a blocking operation on the provided execution context.
// Ensure proper filters since this function expects a single snapshot given the input
func WaitUntilEc2SnapshotCompleted(execCtx context.Context, zone *string, describeSnapshotsInput *ec2.DescribeSnapshotsInput) error {
	// Retry 120 times every 15-16 seconds
	maxDuration := time.Duration(16 * time.Second * 120)
	snapshotsCtx, cancel := context.WithTimeout(execCtx, maxDuration)
	defer cancel()
	awsSdkConfig, err := GetAwsSdkConfig(execCtx, zone)
	if err != nil {
		return err
	}
	describeSnapshotsClient := ec2.NewFromConfig(*awsSdkConfig)
	waiter := *ec2.NewSnapshotCompletedWaiter(describeSnapshotsClient, func(opts *ec2.SnapshotCompletedWaiterOptions) {
		// Max delay of 120 seconds between each attempt is too much because we're defining a custom retry function
		// This will wait 15 seconds between each attempt
		opts.MaxDelay = opts.MinDelay + 1
		opts.Retryable = func(_ context.Context, _ *ec2.DescribeSnapshotsInput, output *ec2.DescribeSnapshotsOutput, err error) (bool, error) {
			// Total failure, stop trying
			if err != nil {
				fmt.Printf("Failed to read snapshot state. error: %v", err)
				return false, err
			}
			// Success
			if output.Snapshots[0].State == awsEc2Types.SnapshotStateCompleted {
				return false, nil
			}
			// Retry, if possible
			return true, nil
		}
	})
	waiter.WaitForOutput(snapshotsCtx, describeSnapshotsInput, maxDuration)
	return nil
}
