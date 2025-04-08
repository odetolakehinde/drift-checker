// Package engine provides the core logic for comparing AWS EC2 instance configurations with their Terraform-defined counterparts.
package engine

import (
	"context"
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
func compareInstances(awsInst, tfInst *common.EC2Instance, filter map[string]bool) common.DriftResult {
	result := common.DriftResult{
		InstanceID:  awsInst.InstanceID,
		Differences: make(map[string]common.FieldDiff),
	}

	// shouldCompare helps to check whether we need to compare an attribute or not
	shouldCompare := func(field string) bool {
		return len(filter) == 0 || filter[field]
	}

	// sort the string fields first
	compareField("instance_type", awsInst.InstanceType, tfInst.InstanceType, filter, result.Differences)
	compareField("image_id", awsInst.ImageID, tfInst.ImageID, filter, result.Differences)
	compareField("key_name", awsInst.KeyName, tfInst.KeyName, filter, result.Differences)
	compareField("subnet_id", awsInst.SubnetID, tfInst.SubnetID, filter, result.Differences)
	compareField("vpc_id", awsInst.VpcID, tfInst.VpcID, filter, result.Differences)
	compareField("iam_instance_profile", awsInst.IamInstanceProfile, tfInst.IamInstanceProfile, filter, result.Differences)
	compareField("monitoring", awsInst.Monitoring, tfInst.Monitoring, filter, result.Differences)
	compareField("architecture", awsInst.Architecture, tfInst.Architecture, filter, result.Differences)
	compareField("virtualization_type", awsInst.VirtualizationType, tfInst.VirtualizationType, filter, result.Differences)

	// then, we do for the tags
	if shouldCompare("tags") && !reflect.DeepEqual(awsInst.Tags, tfInst.Tags) {
		compareMap("tags", awsInst.Tags, tfInst.Tags, filter, result.Differences)
	}

	// time for security groups (SGs)
	if shouldCompare("security_groups") {
		awsSg := append([]string{}, awsInst.SecurityGroups...)
		tfSg := append([]string{}, tfInst.SecurityGroups...)
		sort.Strings(awsSg)
		sort.Strings(tfSg)
		if !slices.Equal(awsSg, tfSg) {
			compareSlice("security_groups", awsSg, tfSg, filter, result.Differences)
		}
	}

	// compare block device mappings
	if shouldCompare("block_device_mappings") {
		awsBdm := common.FlattenBlockDevices(awsInst.BlockDeviceMappings)
		tfBdm := common.FlattenBlockDevices(tfInst.BlockDeviceMappings)
		sort.Strings(awsBdm)
		sort.Strings(tfBdm)
		if !slices.Equal(awsBdm, tfBdm) {
			compareSlice("block_device_mappings", awsBdm, tfBdm, filter, result.Differences)
		}
	}

	result.DriftDetected = len(result.Differences) > 0
	return result
}

// compareField performs a type-safe equality check between two values of comparable type T.
// If the specified field is included in the comparison filter (or no filter is set),
// and the values differ, the difference is added to the output map.
func compareField[T comparable](field string, a, b T, filter map[string]bool, out map[string]common.FieldDiff) {
	if len(filter) > 0 && !filter[field] {
		return
	}

	if a != b {
		out[field] = common.FieldDiff{
			AWS:       a,
			Terraform: b,
		}
	}
}

// compareMap checks for equality between two string maps and adds a diff if they differ.
// This avoids unnecessary drift comparison for fields that are not filtered.
func compareMap(field string, a, b map[string]string, filter map[string]bool, out map[string]common.FieldDiff) {
	if len(filter) > 0 && !filter[field] {
		return
	}
	if !reflect.DeepEqual(a, b) {
		out[field] = common.FieldDiff{AWS: a, Terraform: b}
	}
}

// compareSlice compares two slices of strings for equality regardless of order.
// If the specified field is included in the comparison filter (or no filter is set),
// and the sorted slices differ, the difference is added to the output map.
func compareSlice(field string, a, b []string, filter map[string]bool, out map[string]common.FieldDiff) {
	if len(filter) > 0 && !filter[field] {
		return
	}

	sa := append([]string(nil), a...)
	sb := append([]string(nil), b...)
	sort.Strings(sa)
	sort.Strings(sb)

	if !slices.Equal(sa, sb) {
		out[field] = common.FieldDiff{
			AWS:       sa,
			Terraform: sb,
		}
	}
}

// CompareAllInstances performs concurrent drift comparison between AWS and Terraform EC2 instances.
//
// It uses a bounded worker pool to limit memory and CPU usage,
// and respects context cancellation (e.g., timeouts or user interrupts).
//
// Parameters:
//   - ctx: context to support cancellation
//   - awsInstances: instances fetched from AWS
//   - tfInstances: instances parsed from Terraform state
//   - filter: a map of field names to check for drift
//
// Returns:
//   - []DriftResult containing drift reports per instance (only meaningful results)
func CompareAllInstances(ctx context.Context, awsInstances []*common.EC2Instance, tfInstances []*common.EC2Instance, filter map[string]bool) []common.DriftResult {
	const maxWorkers = 10

	results := make([]common.DriftResult, 0, len(awsInstances))
	resultsCh := make(chan common.DriftResult)
	tasks := make(chan *common.EC2Instance)

	// Build a map for quick lookup
	tfMap := make(map[string]*common.EC2Instance)
	for _, tfInst := range tfInstances {
		tfMap[tfInst.InstanceID] = tfInst
	}

	var wg sync.WaitGroup

	// Start bounded worker pool
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case awsInst, ok := <-tasks:
					if !ok {
						return
					}
					tfInst, ok := tfMap[awsInst.InstanceID]
					if !ok {
						resultsCh <- common.DriftResult{
							InstanceID:    awsInst.InstanceID,
							DriftDetected: true,
							Differences: map[string]common.FieldDiff{
								"terraform_state": {
									AWS:       "exists",
									Terraform: "missing",
								},
							},
						}
						continue
					}

					drift := compareInstances(awsInst, tfInst, filter)
					resultsCh <- drift
				}
			}
		}()
	}

	// Feed tasks
	go func() {
		for _, awsInst := range awsInstances {
			tasks <- awsInst
		}
		close(tasks)
	}()

	// Close results when all workers complete
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	// Collect results
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
