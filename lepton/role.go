package lepton

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

const vmiName = "vmimport"

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

// ResourceWrapper wraps the resource block as it can be 2 types, an
// array of strings or wildcard.
type ResourceWrapper struct {
	Everything bool
	List       []string
}

// UnmarshalJSON is a custom unmarshaller to support multiple types for
// Resource.
func (w *ResourceWrapper) UnmarshalJSON(data []byte) (err error) {
	if string(data) == "\"*\"" {
		w.Everything = true
	} else {
		json.Unmarshal([]byte(data), &w.List)
	}
	return nil
}

// MarshalJSON is a custom unmarshaller to support multiple types for
// Resource.
func (w ResourceWrapper) MarshalJSON() ([]byte, error) {
	if w.Everything {
		return []byte("\"*\""), nil
	}

	return json.Marshal(w.List)
}

// RoleStatement is the representation of a statement of a RolePolicy.
type RoleStatement struct {
	Effect   string
	Action   []string
	Resource ResourceWrapper
}

// RolePolicy is the representation of an aws role policy.
type RolePolicy struct {
	Version   string
	Statement []RoleStatement
}

func (w *ResourceWrapper) updateResource(bucket string) {
	w.List = append(w.List, "arn:aws:s3:::"+bucket)
	w.List = append(w.List, "arn:aws:s3:::"+bucket+"/*")
}

func appendBucket(role string, bucket string) string {
	rp := &RolePolicy{}
	err := json.Unmarshal([]byte(role), rp)
	if err != nil {
		fmt.Println(err)
	}

	for i := 0; i < len(rp.Statement); i++ {
		// wtf is the point of encoding something arbitrary
		actions := rp.Statement[i].Action
		for x := 0; x < len(actions); x++ {
			if actions[x] == "s3:GetObject" {
				rp.Statement[i].Resource.updateResource(bucket)
			}
		}
	}

	b, err := json.Marshal(rp)
	if err != nil {
		fmt.Println(err)
	}

	return string(b)

}

func roleError(bucket string, err error) {
	fmt.Println(err)
	fmt.Println(roleMsg)

	rp := strings.ReplaceAll(rolePolicy, "my-bucket", bucket)

	fmt.Printf("role policy:\n%+v\n", rp)
	fmt.Printf("role :\n%+v\n", vmieDoc)
}

func findBucketInPolicy(svc *iam.IAM, bucket string) (string, error) {

	// find the policy
	lpi := &iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(vmiName),
	}

	pl, err := svc.ListAttachedRolePolicies(lpi)
	if err != nil {
		return "", err
	}

	pid := ""
	for i := 0; i < len(pl.AttachedPolicies); i++ {
		if aws.StringValue(pl.AttachedPolicies[i].PolicyName) == vmiName {
			pid = aws.StringValue(pl.AttachedPolicies[i].PolicyArn)
			break
		}
	}

	if pid == "" {
		return "", errors.New("can't find vmimport policy for " + vmiName)
	}

	gpi := &iam.GetPolicyInput{
		PolicyArn: aws.String(pid),
	}

	po, err := svc.GetPolicy(gpi)
	if err != nil {
		return "", err
	}

	gpvi := &iam.GetPolicyVersionInput{
		PolicyArn: aws.String(pid),
		VersionId: po.Policy.DefaultVersionId,
	}

	pv, err := svc.GetPolicyVersion(gpvi)
	if err != nil {
		return "", err
	}

	s := aws.StringValue(pv.PolicyVersion.Document)

	return url.QueryUnescape(s)

}

// VerifyRole ensures we have a role and attached policy for the vmie service to hit our
// bucket.
func VerifyRole(zone string, bucket string) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(zone)},
	)

	svc := iam.New(sess)

	resp, err := svc.ListRoles(&iam.ListRolesInput{})
	if err != nil {
		roleError(bucket, err)
		os.Exit(1)
	}

	// this is probably a good candidate to cache in a metadata file
	// somewhere - having to do this on each upload is insane
	for _, role := range resp.Roles {
		if aws.StringValue(role.RoleName) == vmiName {

			dval, err := findBucketInPolicy(svc, bucket)
			if err != nil {
				roleError(bucket, err)
				os.Exit(1)
			}

			if strings.Contains(dval, bucket) {
				return
			}

			s := appendBucket(dval, bucket)

			uri := &iam.PutRolePolicyInput{
				PolicyName:     aws.String(vmiName),
				RoleName:       aws.String(vmiName),
				PolicyDocument: aws.String(s),
			}

			_, err = svc.PutRolePolicy(uri)
			if err != nil {
				roleError(bucket, err)
				os.Exit(1)
			}

			return
		}
	}

	err = createRole(svc, bucket)
	if err != nil {
		roleError(bucket, err)
		os.Exit(1)
	}

}

func createRole(svc *iam.IAM, bucket string) error {
	fmt.Println("creating a vmimport role for bucket " + bucket)

	ri := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(vmieDoc),
		RoleName:                 aws.String(vmiName),
	}

	_, err := svc.CreateRole(ri)
	if err != nil {
		return err
	}

	rp := strings.ReplaceAll(rolePolicy, "my-bucket", bucket)

	cpi := &iam.CreatePolicyInput{
		PolicyDocument: aws.String(rp),
		PolicyName:     aws.String(vmiName),
	}

	policyOut, err := svc.CreatePolicy(cpi)
	if err != nil {
		return err
	}

	ari := &iam.AttachRolePolicyInput{
		PolicyArn: policyOut.Policy.Arn,
		RoleName:  aws.String(vmiName),
	}

	_, err = svc.AttachRolePolicy(ari)
	if err != nil {
		return err
	}

	return nil
}
