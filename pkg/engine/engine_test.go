package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/odetolakehinde/drift-checker/pkg/common"
)

func TestCompareInstances(t *testing.T) {
	awsBase := &common.EC2Instance{
		InstanceID:          "i-1",
		InstanceType:        "t3.micro",
		ImageID:             "ami-1",
		KeyName:             "keypair",
		Monitoring:          true,
		Tags:                map[string]string{"Name": "web"},
		SecurityGroups:      []string{"sg-1", "sg-2"},
		BlockDeviceMappings: []common.BlockDeviceMapping{{DeviceName: "/dev/sda1", VolumeID: "vol-1"}},
	}

	tests := []struct {
		name     string
		aws      *common.EC2Instance
		tf       *common.EC2Instance
		filter   map[string]bool
		wantDiff map[string]common.FieldDiff
	}{
		{
			name:     "identical instances",
			aws:      awsBase,
			tf:       cloneInstance(awsBase),
			filter:   nil,
			wantDiff: map[string]common.FieldDiff{},
		},
		{
			name: "difference in instance_type",
			aws:  awsBase,
			tf: &common.EC2Instance{
				InstanceID:   "i-1",
				InstanceType: "t2.micro",
			},
			filter: map[string]bool{"instance_type": true},
			wantDiff: map[string]common.FieldDiff{
				"instance_type": {AWS: "t3.micro", Terraform: "t2.micro"},
			},
		},
		{
			name: "difference in tags and block_device_mappings",
			aws:  awsBase,
			tf: &common.EC2Instance{
				InstanceID: "i-1",
				Tags:       map[string]string{"Env": "prod"},
				BlockDeviceMappings: []common.BlockDeviceMapping{
					{DeviceName: "/dev/sda1", VolumeID: "vol-2"},
				},
			},
			filter: map[string]bool{"tags": true, "block_device_mappings": true},
			wantDiff: map[string]common.FieldDiff{
				"tags": {
					AWS:       map[string]string{"Name": "web"},
					Terraform: map[string]string{"Env": "prod"},
				},
				"block_device_mappings": {
					AWS:       []string{"/dev/sda1|vol-1"},
					Terraform: []string{"/dev/sda1|vol-2"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareInstances(tt.aws, tt.tf, tt.filter)

			if !reflect.DeepEqual(got.Differences, tt.wantDiff) {
				t.Errorf("CompareInstances() differences = %v, want %v", got.Differences, tt.wantDiff)
			}
		})
	}
}

func TestCompareAllInstances(t *testing.T) {
	aws := []*common.EC2Instance{
		{InstanceID: "i-1", InstanceType: "t3.micro"},
		{InstanceID: "i-2", InstanceType: "t3.medium"},
	}
	tf := []*common.EC2Instance{
		{InstanceID: "i-1", InstanceType: "t3.micro"},
		{InstanceID: "i-2", InstanceType: "t2.medium"},
	}

	filter := map[string]bool{"instance_type": true}

	results := CompareAllInstances(aws, tf, filter)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for _, r := range results {
		switch r.InstanceID {
		case "i-1":
			if r.DriftDetected {
				t.Errorf("expected no drift for i-1")
			}
		case "i-2":
			if !r.DriftDetected || r.Differences["instance_type"].Terraform != "t2.medium" {
				t.Errorf("expected drift for i-2 with instance_type diff")
			}
		default:
			t.Errorf("unexpected instance ID: %s", r.InstanceID)
		}
	}
}

func TestCloneInstance(t *testing.T) {
	original := &common.EC2Instance{
		InstanceID:     "i-123",
		Tags:           map[string]string{"Name": "test"},
		SecurityGroups: []string{"sg-1", "sg-2"},
		BlockDeviceMappings: []common.BlockDeviceMapping{
			{DeviceName: "/dev/xvda", VolumeID: "vol-1"},
		},
	}

	clone := cloneInstance(original)

	// Should be equal in value
	if !reflect.DeepEqual(original, clone) {
		t.Errorf("cloneInstance: expected deep equal copy, got differences")
	}

	// Should not be the same memory for nested fields
	clone.Tags["Name"] = "changed"
	clone.SecurityGroups[0] = "sg-x"
	clone.BlockDeviceMappings[0].VolumeID = "vol-x"

	if original.Tags["Name"] == "changed" {
		t.Errorf("Tags were mutated in original")
	}
	if original.SecurityGroups[0] == "sg-x" {
		t.Errorf("SecurityGroups were mutated in original")
	}
	if original.BlockDeviceMappings[0].VolumeID == "vol-x" {
		t.Errorf("BlockDeviceMappings were mutated in original")
	}
}

func TestPrintDriftReport_JSON(t *testing.T) {
	result := common.DriftResult{
		InstanceID:    "i-123",
		DriftDetected: true,
		Differences: map[string]common.FieldDiff{
			"instance_type": {AWS: "t3.micro", Terraform: "t2.micro"},
		},
	}

	output := captureOutput(func() {
		PrintDriftReport(result, true)
	})

	var parsed common.DriftResult
	err := json.Unmarshal([]byte(output), &parsed)
	if err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if parsed.InstanceID != "i-123" || !parsed.DriftDetected {
		t.Errorf("unexpected JSON output: %+v", parsed)
	}
}

func TestPrintDriftReport_Human_NoDrift(t *testing.T) {
	result := common.DriftResult{
		InstanceID:    "i-999",
		DriftDetected: false,
		Differences:   map[string]common.FieldDiff{},
	}

	output := captureOutput(func() {
		PrintDriftReport(result, false)
	})

	if !strings.Contains(output, "✅ No drift detected") {
		t.Errorf("expected no drift message, got:\n%s", output)
	}
}

func TestPrintDriftReport_Human_WithDrift(t *testing.T) {
	result := common.DriftResult{
		InstanceID:    "i-456",
		DriftDetected: true,
		Differences: map[string]common.FieldDiff{
			"tags": {AWS: map[string]string{"Name": "A"}, Terraform: map[string]string{"Name": "B"}},
		},
	}

	output := captureOutput(func() {
		PrintDriftReport(result, false)
	})

	if !strings.Contains(output, "Drift Report for Instance ID: i-456") ||
		!strings.Contains(output, "tags:") ||
		!strings.Contains(output, "AWS:") {
		t.Errorf("expected drift fields in output, got:\n%s", output)
	}
}

func TestCaptureOutput_CapturesStdout(t *testing.T) {
	expected := "Hello, test output!"

	output := captureOutput(func() {
		fmt.Println(expected)
	})

	// Trim newline for exact match
	output = strings.TrimSpace(output)

	if output != expected {
		t.Errorf("captureOutput failed: got %q, want %q", output, expected)
	}
}

func cloneInstance(i *common.EC2Instance) *common.EC2Instance {
	cp := *i
	cp.Tags = map[string]string{}
	for k, v := range i.Tags {
		cp.Tags[k] = v
	}
	cp.SecurityGroups = append([]string{}, i.SecurityGroups...)
	cp.BlockDeviceMappings = append([]common.BlockDeviceMapping{}, i.BlockDeviceMappings...)
	return &cp
}

func captureOutput(f func()) string {
	// Redirect stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run function
	f()

	// Restore stdout and capture
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}
