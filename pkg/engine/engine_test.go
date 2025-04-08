package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/odetolakehinde/drift-checker/pkg/common"
)

func TestCompareField(t *testing.T) {
	out := make(map[string]common.FieldDiff)
	compareField("instance_type", "t2.micro", "t2.small", nil, out)

	assert.Len(t, out, 1)
	assert.Equal(t, "t2.micro", out["instance_type"].AWS)
	assert.Equal(t, "t2.small", out["instance_type"].Terraform)
}

func TestCompareField_Equal(t *testing.T) {
	out := make(map[string]common.FieldDiff)
	compareField("instance_type", "t2.micro", "t2.micro", nil, out)
	assert.Empty(t, out)
}

func TestCompareSlice(t *testing.T) {
	out := make(map[string]common.FieldDiff)
	a := []string{"sg-123", "sg-456"}
	b := []string{"sg-456", "sg-999"}

	compareSlice("security_groups", a, b, nil, out)

	assert.Len(t, out, 1)
	assert.ElementsMatch(t, out["security_groups"].AWS.([]string), a)
	assert.ElementsMatch(t, out["security_groups"].Terraform.([]string), b)
}

func TestCompareSlice_Equal(t *testing.T) {
	out := make(map[string]common.FieldDiff)

	compareSlice("security_groups", []string{"sg-1", "sg-2"}, []string{"sg-2", "sg-1"}, nil, out)

	assert.Empty(t, out)
}

func TestCompareSlice_Diff(t *testing.T) {
	out := make(map[string]common.FieldDiff)

	compareSlice("security_groups", []string{"sg-1", "sg-2"}, []string{"sg-2", "sg-3"}, nil, out)

	assert.Contains(t, out, "security_groups")
	assert.ElementsMatch(t, out["security_groups"].AWS.([]string), []string{"sg-1", "sg-2"})
	assert.ElementsMatch(t, out["security_groups"].Terraform.([]string), []string{"sg-2", "sg-3"})
}

func TestCompareSlice_Empty(t *testing.T) {
	out := make(map[string]common.FieldDiff)

	compareSlice("security_groups", []string{}, []string{}, nil, out)

	assert.Empty(t, out)
}

func TestCompareSlice_Filtered(t *testing.T) {
	out := make(map[string]common.FieldDiff)

	filter := map[string]bool{"instance_type": true} // does not include this field
	compareSlice("security_groups", []string{"sg-1"}, []string{"sg-2"}, filter, out)

	assert.Empty(t, out)
}
func TestCompareMap_DiffDetected(t *testing.T) {
	a := map[string]string{"Name": "redis", "Env": "prod"}
	b := map[string]string{"Name": "redis", "Env": "dev"}

	out := make(map[string]common.FieldDiff)
	compareMap("tags", a, b, nil, out)

	assert.Len(t, out, 1)
	assert.Contains(t, out, "tags")
	assert.True(t, reflect.DeepEqual(a, out["tags"].AWS))
	assert.True(t, reflect.DeepEqual(b, out["tags"].Terraform))
}

func TestCompareMap_NoDiff(t *testing.T) {
	a := map[string]string{"Name": "redis"}
	b := map[string]string{"Name": "redis"}

	out := make(map[string]common.FieldDiff)
	compareMap("tags", a, b, nil, out)

	assert.Len(t, out, 0)
}

func TestCompareMap_FilteredOut(t *testing.T) {
	a := map[string]string{"Env": "prod"}
	b := map[string]string{"Env": "staging"}

	out := make(map[string]common.FieldDiff)
	compareMap("tags", a, b, map[string]bool{"instance_type": true}, out)

	assert.Len(t, out, 0)
}

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
			got := compareInstances(tt.aws, tt.tf, tt.filter)

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

	results := CompareAllInstances(context.Background(), aws, tf, filter)

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

func TestCompareAllInstances_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	results := CompareAllInstances(ctx, []*common.EC2Instance{
		{InstanceID: "i-123"},
	}, []*common.EC2Instance{}, nil)

	assert.LessOrEqual(t, len(results), 1)
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
