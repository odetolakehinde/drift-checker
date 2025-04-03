// Package terraform to parse terraform file
package terraform

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/odetolakehinde/drift-checker/pkg/common"
)

// ParseTerraformState parses a Terraform state file and extracts EC2Instance values.
func ParseTerraformState(stateFilePath string) ([]*common.EC2Instance, error) {
	data, err := os.ReadFile(stateFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state common.TerraformState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state file: %w", err)
	}

	var instances []*common.EC2Instance

	for _, res := range state.Resources {
		if res.Type != "aws_instance" {
			continue
		}

		for _, inst := range res.Instances {
			attr := inst.Attributes

			ec2Inst := &common.EC2Instance{
				InstanceID:          common.ToString(attr["id"]),
				InstanceType:        common.ToString(attr["instance_type"]),
				ImageID:             common.ToString(attr["ami"]),
				KeyName:             common.ToString(attr["key_name"]),
				AvailabilityZone:    common.ToString(attr["availability_zone"]),
				PrivateIPAddress:    common.ToString(attr["private_ip"]),
				PublicIPAddress:     common.ToString(attr["public_ip"]),
				SubnetID:            common.ToString(attr["subnet_id"]),
				IamInstanceProfile:  common.ToString(attr["iam_instance_profile"]),
				Monitoring:          common.ToBool(attr["monitoring"]),
				Architecture:        common.ToString(attr["architecture"]),
				VirtualizationType:  common.ToString(attr["virtualization_type"]),
				Tags:                common.ConvertToStringMap(attr["tags"]),
				SecurityGroups:      common.ConvertToStringSlice(attr["vpc_security_group_ids"]),
				BlockDeviceMappings: common.ExtractBlockDevices(attr["root_block_device"]),
			}

			instances = append(instances, ec2Inst)
		}
	}

	return instances, nil
}
