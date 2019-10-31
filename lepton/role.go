package lepton

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

var vmieDoc = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "vmie.amazonaws.com"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringEquals": {
          "sts:Externalid": "vmimport"
        }
      }
    }
  ]
}
`

var rolePolicy = `{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "s3:GetBucketLocation",
                "s3:GetObject",
                "s3:ListBucket"
            ],
            "Resource": [
                "arn:aws:s3:::my-bucket",
                "arn:aws:s3:::my-bucket/*"
            ]
        },
        {
            "Effect": "Allow",
            "Action": [
                "ec2:ModifySnapshotAttribute",
                "ec2:CopySnapshot",
                "ec2:RegisterImage",
                "ec2:Describe*"
            ],
            "Resource": "*"
        }
    ]
}
`

var roleMsg = `We could not create a vmimport Role for you as this account probably does not have the right permissions. Please ask or manually create a vmimport role. For more information see: https://docs.aws.amazon.com/vm-import/latest/userguide/vmimport-troubleshooting.html`

func roleError(bucket string) {
	fmt.Println(roleMsg)

	rp := strings.ReplaceAll(rolePolicy, "my-bucket", bucket)

	fmt.Printf("role policy:\n%+v\n", rp)
	fmt.Printf("role :\n%+v\n", vmieDoc)
}

// VerifyRole ensures we have a role for the vmie service to hit our
// bucket.
func VerifyRole(ctx *Context, bucket string) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(ctx.config.CloudConfig.Zone)},
	)

	svc := iam.New(sess)

	resp, err := svc.ListRoles(&iam.ListRolesInput{})
	if err != nil {
		fmt.Println("error listing roles:", err)
	}

	for _, role := range resp.Roles {
		if aws.StringValue(role.RoleName) == "vmimport" {
			return
		}
	}

	fmt.Println("creating a vmimport role for bucket " + bucket)

	ri := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(vmieDoc),
		RoleName:                 aws.String("vmimport"),
	}

	_, err = svc.CreateRole(ri)
	if err != nil {
		fmt.Println(err)
		roleError(bucket)
		os.Exit(1)
	}

	rp := strings.ReplaceAll(rolePolicy, "my-bucket", bucket)

	cpi := &iam.CreatePolicyInput{
		PolicyDocument: aws.String(rp),
		PolicyName:     aws.String("vmimport"),
	}

	policyOut, err := svc.CreatePolicy(cpi)
	if err != nil {
		fmt.Println(err)
		roleError(bucket)
		os.Exit(1)
	}

	ari := &iam.AttachRolePolicyInput{
		PolicyArn: policyOut.Policy.Arn,
		RoleName:  aws.String("vmimport"),
	}

	_, err = svc.AttachRolePolicy(ari)
	if err != nil {
		fmt.Println(err)
		roleError(bucket)
		os.Exit(1)
	}

}
