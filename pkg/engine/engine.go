// Package engine provides the core logic for comparing AWS EC2 instance configurations with their Terraform-defined counterparts.
package engine

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/odetolakehinde/drift-checker/pkg/common"
)

// CompareInstances detects drift between an EC2 instance from AWS and Terraform state.
func CompareInstances(awsInst, tfInst *common.EC2Instance, filter map[string]bool) common.DriftResult {
	result := common.DriftResult{
		InstanceID:  awsInst.InstanceID,
		Differences: make(map[string]common.FieldDiff),
	}

	// shouldCompare helps to check whether we need to compare an attribute or not
	shouldCompare := func(field string) bool {
		return len(filter) == 0 || filter[field]
	}

	compare := func(field string, a, b any) {
		if shouldCompare(field) && !reflect.DeepEqual(a, b) {
			result.Differences[field] = common.FieldDiff{
				AWS:       a,
				Terraform: b,
			}
		}
	}

	// sort the string fields first
	compare("instance_type", awsInst.InstanceType, tfInst.InstanceType)
	compare("image_id", awsInst.ImageID, tfInst.ImageID)
	compare("key_name", awsInst.KeyName, tfInst.KeyName)
	compare("subnet_id", awsInst.SubnetID, tfInst.SubnetID)
	compare("vpc_id", awsInst.VpcID, tfInst.VpcID)
	compare("iam_instance_profile", awsInst.IamInstanceProfile, tfInst.IamInstanceProfile)
	compare("monitoring", awsInst.Monitoring, tfInst.Monitoring)
	compare("architecture", awsInst.Architecture, tfInst.Architecture)
	compare("virtualization_type", awsInst.VirtualizationType, tfInst.VirtualizationType)

	// then, we do for the tags
	if shouldCompare("tags") && !reflect.DeepEqual(awsInst.Tags, tfInst.Tags) {
		compare("tags", awsInst.Tags, tfInst.Tags)
	}

	// time for security groups (SGs)
	if shouldCompare("security_groups") {
		awsSg := append([]string{}, awsInst.SecurityGroups...)
		tfSg := append([]string{}, tfInst.SecurityGroups...)
		sort.Strings(awsSg)
		sort.Strings(tfSg)
		if !slices.Equal(awsSg, tfSg) {
			compare("security_groups", awsSg, tfSg)
		}
	}

	// compare block device mappings
	if shouldCompare("block_device_mappings") {
		awsBdm := common.FlattenBlockDevices(awsInst.BlockDeviceMappings)
		tfBdm := common.FlattenBlockDevices(tfInst.BlockDeviceMappings)
		sort.Strings(awsBdm)
		sort.Strings(tfBdm)
		if !slices.Equal(awsBdm, tfBdm) {
			compare("block_device_mappings", awsBdm, tfBdm)
		}
	}

	result.DriftDetected = len(result.Differences) > 0
	return result
}

// CompareAllInstances compares multiple EC2 instances from AWS and Terraform concurrently.
func CompareAllInstances(awsInstances []*common.EC2Instance, tfInstances []*common.EC2Instance, filter map[string]bool) []common.DriftResult {
	results := make([]common.DriftResult, 0)
	resultsCh := make(chan common.DriftResult)
	var wg sync.WaitGroup

	// first, nice to build a map from Terraform by InstanceID for quick lookup
	tfMap := make(map[string]*common.EC2Instance)
	for _, tfInst := range tfInstances {
		tfMap[tfInst.InstanceID] = tfInst
	}

	// then, we compare each AWS instance against its Terraform counterpart
	for _, awsInst := range awsInstances {
		tfInst, ok := tfMap[awsInst.InstanceID]
		if !ok {
			// Instance exists in AWS but not in Terraform
			results = append(results, common.DriftResult{
				InstanceID:    awsInst.InstanceID,
				DriftDetected: true,
				Differences: map[string]common.FieldDiff{
					"terraform": {
						AWS:       "exists",
						Terraform: "missing",
					},
				},
			})
			continue
		}

		wg.Add(1)
		go func(aInst *common.EC2Instance, tInst *common.EC2Instance) {
			defer wg.Done()
			result := CompareInstances(aInst, tInst, filter)
			resultsCh <- result
		}(awsInst, tfInst)
	}

	// safe to close the channel once all goroutines are done
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	for r := range resultsCh {
		results = append(results, r)
	}

	return results
}

// PrintDriftReport outputs a drift result to standard output in either
// human-readable or JSON format, depending on the asJSON flag.
//
// If asJSON is true, the result is printed as pretty-formatted JSON.
// Otherwise, a structured plain-text report is printed, showing which
// fields differ between AWS and Terraform for the given EC2 instance.
func PrintDriftReport(result common.DriftResult, asJSON bool) {
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			log.Printf("failed to encode drift report: %v\n", err)
		}

		// let's also write to drift_<instance-id>_timestamp.json
		fileName := fmt.Sprintf("results/drift_%s_%d.json", result.InstanceID, time.Now().Unix())
		f, err := os.Create(fileName)
		if err != nil {
			log.Printf("❌ failed to write drift JSON to file: %v", err)
			return
		}
		defer func(f *os.File) {
			err = f.Close()
			if err != nil {
				return
			}
		}(f)

		encFile := json.NewEncoder(f)
		encFile.SetIndent("", "  ")
		if err := encFile.Encode(result); err != nil {
			log.Printf("❌failed to encode to file: %v\n", err)
		} else {
			fmt.Printf("JSON drift report written to: %s\n", fileName)
		}

		return
	}

	header := fmt.Sprintf("Drift Report for Instance ID: %s", result.InstanceID)
	fmt.Println(strings.Repeat("=", len(header)))
	fmt.Println(header)
	fmt.Println(strings.Repeat("=", len(header)))

	if !result.DriftDetected {
		fmt.Println("✅ No drift detected.")
		return
	}

	fmt.Println("❌ Drift detected in the following fields:")
	fmt.Println()

	for field, diff := range result.Differences {
		fmt.Printf("- %s:\n", field)
		fmt.Printf("    AWS:       %v\n", diff.AWS)
		fmt.Printf("    Terraform: %v\n", diff.Terraform)
		fmt.Println()
	}
}
