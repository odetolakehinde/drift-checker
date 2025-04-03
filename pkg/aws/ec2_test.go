package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"

	"github.com/odetolakehinde/drift-checker/pkg/common"
)

type mockEC2Client struct {
	output *ec2.DescribeInstancesOutput
	err    error
}

func (m *mockEC2Client) DescribeInstances(ctx context.Context, input *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return m.output, m.err
}

func TestGetInstanceFromClient_Success(t *testing.T) {
	client := &mockEC2Client{
		output: &ec2.DescribeInstancesOutput{
			Reservations: []ec2Types.Reservation{
				{
					Instances: []ec2Types.Instance{
						{
							InstanceId:       common.GetStringPointer("i-123"),
							InstanceType:     ec2Types.InstanceTypeT2Micro,
							ImageId:          common.GetStringPointer("ami-abc"),
							KeyName:          common.GetStringPointer("my-key"),
							PrivateIpAddress: common.GetStringPointer("10.0.0.1"),
							PublicIpAddress:  common.GetStringPointer("3.3.3.3"),
							Monitoring:       &ec2Types.Monitoring{State: ec2Types.MonitoringStateEnabled},
							Tags: []ec2Types.Tag{
								{Key: common.GetStringPointer("Name"), Value: common.GetStringPointer("TestInstance")},
							},
							State: &ec2Types.InstanceState{
								Code: nil,
								Name: "running",
							},
							Architecture:       ec2Types.ArchitectureValuesX8664,
							VirtualizationType: ec2Types.VirtualizationTypeHvm,
							BlockDeviceMappings: []ec2Types.InstanceBlockDeviceMapping{
								{
									DeviceName: common.GetStringPointer("/dev/xvda"),
									Ebs: &ec2Types.EbsInstanceBlockDevice{
										VolumeId: common.GetStringPointer("vol-001"),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := GetInstanceFromClient(context.Background(), client, "i-123")
	assert.NoError(t, err)
	assert.Equal(t, "i-123", result.InstanceID)
	assert.Equal(t, "t2.micro", result.InstanceType)
	assert.Equal(t, "ami-abc", result.ImageID)
	assert.Equal(t, "TestInstance", result.Tags["Name"])
	assert.Equal(t, "vol-001", result.BlockDeviceMappings[0].VolumeID)
	assert.Equal(t, "running", result.State)
}

func TestGetInstanceFromClient_NotFound(t *testing.T) {
	client := &mockEC2Client{
		output: &ec2.DescribeInstancesOutput{
			Reservations: []ec2Types.Reservation{},
		},
	}

	_, err := GetInstanceFromClient(context.Background(), client, "i-missing")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetInstanceFromClient_DescribeError(t *testing.T) {
	client := &mockEC2Client{
		output: nil,
		err:    assert.AnError,
	}

	_, err := GetInstanceFromClient(context.Background(), client, "i-err")
	assert.ErrorIs(t, err, assert.AnError)
}
