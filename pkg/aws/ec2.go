// Package aws to interact w AWS resources
package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/odetolakehinde/drift-checker/pkg/common"
)

// EC2Client defines the subset of AWS EC2 methods used by this application.
type EC2Client interface {
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
}

// GetInstance retrieves the configuration of an EC2 instance by its ID.
func GetInstance(ctx context.Context, instanceID string) (*common.EC2Instance, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	client := ec2.NewFromConfig(cfg)
	return GetInstanceFromClient(ctx, client, instanceID)
}

// GetInstanceFromClient retrieves the configuration of a specific EC2 instance
func GetInstanceFromClient(ctx context.Context, client EC2Client, instanceID string) (*common.EC2Instance, error) {
	output, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return nil, fmt.Errorf("describe instances: %w", err)
	}

	if len(output.Reservations) == 0 || len(output.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("instance %s not found", instanceID)
	}

	//export AWS_ACCESS_KEY_ID=AKIAZG22WNABKUL6YBFY
	//export AWS_SECRET_ACCESS_KEY=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
	//export AWS_REGION=us-east-2

	instance := output.Reservations[0].Instances[0]

	// extract security groups (using GroupId as an identifier)
	var sgIDs []string
	for _, sg := range instance.SecurityGroups {
		if sg.GroupId != nil {
			sgIDs = append(sgIDs, *sg.GroupId)
		}
	}

	// Extract tags into a map for easier comparison.
	tags := make(map[string]string)
	for _, tag := range instance.Tags {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}

	// Extract block device mappings.
	var bdms []common.BlockDeviceMapping
	for _, mapping := range instance.BlockDeviceMappings {
		var volumeID string
		if mapping.Ebs != nil && mapping.Ebs.VolumeId != nil {
			volumeID = *mapping.Ebs.VolumeId
		}
		if mapping.DeviceName != nil {
			bdms = append(bdms, common.BlockDeviceMapping{
				DeviceName: *mapping.DeviceName,
				VolumeID:   volumeID,
			})
		}
	}

	// Check IAM Instance Profile if exists.
	var iamProfile string
	if instance.IamInstanceProfile != nil && instance.IamInstanceProfile.Arn != nil {
		iamProfile = *instance.IamInstanceProfile.Arn
	}

	// Determine if detailed monitoring is enabled.
	monitoringEnabled := false
	if instance.Monitoring != nil && instance.Monitoring.State != "" {
		monitoringEnabled = instance.Monitoring.State == "enabled"
	}

	// Determine the availability zone.
	availabilityZone := ""
	if instance.Placement != nil && instance.Placement.AvailabilityZone != nil {
		availabilityZone = *instance.Placement.AvailabilityZone
	}

	ec2Inst := &common.EC2Instance{
		InstanceID:          common.GetString(instance.InstanceId),
		InstanceType:        string(instance.InstanceType),
		ImageID:             common.GetString(instance.ImageId),
		KeyName:             common.GetString(instance.KeyName),
		State:               string(instance.State.Name),
		AvailabilityZone:    availabilityZone,
		PrivateIPAddress:    common.GetString(instance.PrivateIpAddress),
		PublicIPAddress:     common.GetString(instance.PublicIpAddress),
		SubnetID:            common.GetString(instance.SubnetId),
		VpcID:               common.GetString(instance.VpcId),
		SecurityGroups:      sgIDs,
		Tags:                tags,
		BlockDeviceMappings: bdms,
		IamInstanceProfile:  iamProfile,
		Monitoring:          monitoringEnabled,
		Architecture:        string(instance.Architecture),
		VirtualizationType:  string(instance.VirtualizationType),
	}

	return ec2Inst, nil
}
