package terraform

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/odetolakehinde/drift-checker/pkg/common"
)

func TestParseTerraformState(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantErr  bool
		wantInst []*common.EC2Instance
	}{
		{
			name: "valid aws_instance",
			content: `{
				"resources": [
					{
						"type": "aws_instance",
						"name": "example",
						"instances": [
							{
								"attributes": {
									"id": "i-abc123",
									"ami": "ami-xyz",
									"instance_type": "t3.micro",
									"key_name": "my-key",
									"availability_zone": "us-east-1a",
									"private_ip": "10.0.0.1",
									"public_ip": "3.3.3.3",
									"subnet_id": "subnet-123",
									"vpc_security_group_ids": ["sg-123"],
									"iam_instance_profile": "test-profile",
									"monitoring": true,
									"architecture": "x86_64",
									"virtualization_type": "hvm",
									"tags": {
										"Name": "test",
										"Env": "dev"
									},
									"root_block_device": [
										{
											"device_name": "/dev/xvda",
											"volume_id": "vol-0123"
										}
									]
								}
							}
						]
					}
				]
			}`,
			wantErr: false,
			wantInst: []*common.EC2Instance{
				{
					InstanceID:         "i-abc123",
					InstanceType:       "t3.micro",
					ImageID:            "ami-xyz",
					KeyName:            "my-key",
					AvailabilityZone:   "us-east-1a",
					PrivateIpAddress:   "10.0.0.1",
					PublicIpAddress:    "3.3.3.3",
					SubnetID:           "subnet-123",
					IamInstanceProfile: "test-profile",
					Monitoring:         true,
					Architecture:       "x86_64",
					VirtualizationType: "hvm",
					Tags:               map[string]string{"Name": "test", "Env": "dev"},
					SecurityGroups:     []string{"sg-123"},
					BlockDeviceMappings: []common.BlockDeviceMapping{
						{DeviceName: "/dev/xvda", VolumeID: "vol-0123"},
					},
				},
			},
		},
		{
			name:    "invalid json",
			content: `{not valid`,
			wantErr: true,
		},
		{
			name:     "no aws_instance resources",
			content:  `{"resources":[{"type":"aws_s3_bucket","name":"bucket","instances":[]}]}`,
			wantErr:  false,
			wantInst: nil, // no instance extracted
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "tfstate-test-*.tfstate")
			assert.NoError(t, err)
			defer func(name string) {
				_ = os.Remove(name)
			}(tmpFile.Name())

			_, err = tmpFile.WriteString(tt.content)
			assert.NoError(t, err)
			err = tmpFile.Close()
			if err != nil {
				return
			}

			got, err := ParseTerraformState(tmpFile.Name())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantInst, got)
			}
		})
	}
}
