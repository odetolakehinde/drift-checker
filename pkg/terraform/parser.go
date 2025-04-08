// Package terraform to parse terraform file
package terraform

import (
	"context"
	"encoding/json"
	"os"

	"github.com/rs/zerolog"

	"github.com/odetolakehinde/drift-checker/pkg/common"
)

// Parser defines a facade for Terraform state parsing.
type Parser interface {
	Load(path string) ([]*common.EC2Instance, error)
}

type stateParser struct {
	logger zerolog.Logger
}

func NewParser(_ context.Context, logger zerolog.Logger) Parser {
	return &stateParser{
		logger: logger.With().Str(common.LogStrLayer, "terraform").Logger(),
	}
}

func (p *stateParser) Load(path string) ([]*common.EC2Instance, error) {
	log := p.logger.With().
		Str(common.LogStrMethod, "Load - parseTerraformState").
		Str("path", path).
		Logger()
	return parseTerraformState(log, path)
}

// parseTerraformState parses a Terraform state file and extracts EC2Instance values.
func parseTerraformState(log zerolog.Logger, stateFilePath string) ([]*common.EC2Instance, error) {
	data, err := os.ReadFile(stateFilePath)
	if err != nil {
		log.Err(err).Msg("failed to read state file")
		return nil, common.ErrStateFileNotProvided
	}

	var state common.TerraformState
	if err = json.Unmarshal(data, &state); err != nil {
		log.Err(err).Msg("unable to marshal or parse state file - it is invalid")
		return nil, common.ErrInvalidStateFile
	}

	var instances []*common.EC2Instance

	if len(state.Resources) == 0 {
		log.Err(err).Msg("no terraform resources found in state file")
		return nil, common.ErrTerraformInstanceMissing
	}

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
