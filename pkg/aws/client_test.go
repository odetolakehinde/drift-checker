package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/odetolakehinde/drift-checker/pkg/common"
)

// mockEC2Client implements aws.EC2Client
type mockEC2Client struct {
	output *ec2.DescribeInstancesOutput
	err    error
}

func (m *mockEC2Client) DescribeInstances(_ context.Context, _ *ec2.DescribeInstancesInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return m.output, m.err
}

func TestGetInstanceFromClient_Success(t *testing.T) {
	client := &mockEC2Client{
		output: &ec2.DescribeInstancesOutput{
			Reservations: []ec2Types.Reservation{
				{
					Instances: []ec2Types.Instance{
						{
							InstanceId:   common.GetStringPointer("i-1234567890abcdef0"),
							InstanceType: ec2Types.InstanceTypeT2Micro,
							ImageId:      common.GetStringPointer("ami-123"),
							KeyName:      common.GetStringPointer("test-key"),
							Monitoring:   &ec2Types.Monitoring{State: "disabled"},
							Tags: []ec2Types.Tag{
								{Key: common.GetStringPointer("Name"), Value: common.GetStringPointer("test-instance")},
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
									Ebs:        &ec2Types.EbsInstanceBlockDevice{VolumeId: common.GetStringPointer("vol-abc123")},
								},
							},
						},
					},
				},
			},
		},
	}

	svc := &ec2Service{
		logger: zerolog.Nop(), // use silent logger
	}

	result, err := svc.GetInstanceFromClient(context.Background(), client, "i-1234567890abcdef0")

	assert.NoError(t, err)
	assert.Equal(t, "i-1234567890abcdef0", result.InstanceID)
	assert.Equal(t, "t2.micro", result.InstanceType)
	assert.Equal(t, "ami-123", result.ImageID)
	assert.Equal(t, "test-key", result.KeyName)
	assert.Equal(t, "vol-abc123", result.BlockDeviceMappings[0].VolumeID)
	assert.Equal(t, "test-instance", result.Tags["Name"])
}

func TestGetInstanceFromClient_NotFound(t *testing.T) {
	client := &mockEC2Client{
		output: &ec2.DescribeInstancesOutput{
			Reservations: []ec2Types.Reservation{},
		},
	}

	svc := &ec2Service{
		logger: zerolog.Nop(), // use silent logger
	}

	_, err := svc.GetInstanceFromClient(context.Background(), client, "i-missing")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetInstanceFromClient_DescribeError(t *testing.T) {
	client := &mockEC2Client{
		output: nil,
		err:    assert.AnError,
	}

	svc := &ec2Service{
		logger: zerolog.Nop(), // use silent logger
	}

	_, err := svc.GetInstanceFromClient(context.Background(), client, "i-err")
	assert.ErrorIs(t, err, common.ErrAWSDescribeFailure)
}
