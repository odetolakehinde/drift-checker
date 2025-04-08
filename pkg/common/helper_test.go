package common

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

var (
	text = "ninja is at it again"
)

func TestGetString(t *testing.T) {
	tests := []struct {
		name string
		in   *string
		want string
	}{
		{"non-nil", &text, text},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetString(tt.in)
			if got != tt.want {
				t.Errorf("GetString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want string
	}{
		{"string", text, text},
		{"int", 42, ""},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToString(tt.in); got != tt.want {
				t.Errorf("ToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToBool(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want bool
	}{
		{"bool true", true, true},
		{"bool false", false, false},
		{"string input", "true", false},
		{"nil input", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToBool(tt.in); got != tt.want {
				t.Errorf("ToBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertToStringMap(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want map[string]string
	}{
		{
			"valid map",
			map[string]interface{}{"Name": "web", "Env": "prod"},
			map[string]string{"Name": "web", "Env": "prod"},
		},
		{
			"not a map",
			[]int{1, 2},
			map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertToStringMap(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertToStringMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertToStringSlice(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want []string
	}{
		{
			"valid slice",
			[]interface{}{"a", "b", "c"},
			[]string{"a", "b", "c"},
		},
		{
			"mixed types",
			[]interface{}{"a", 123, true},
			[]string{"a", "123", "true"},
		},
		{
			"not a slice",
			map[string]string{"x": "y"},
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertToStringSlice(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertToStringSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlattenBlockDevices(t *testing.T) {
	tests := []struct {
		name string
		in   []BlockDeviceMapping
		want []string
	}{
		{
			"one device",
			[]BlockDeviceMapping{{DeviceName: "/dev/xvda", VolumeID: "vol-123"}},
			[]string{"/dev/xvda|vol-123"},
		},
		{
			"multiple devices",
			[]BlockDeviceMapping{
				{DeviceName: "/dev/xvda", VolumeID: "vol-1"},
				{DeviceName: "/dev/xvdb", VolumeID: "vol-2"},
			},
			[]string{"/dev/xvda|vol-1", "/dev/xvdb|vol-2"},
		},
		{
			"empty input",
			nil,
			[]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FlattenBlockDevices(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FlattenBlockDevices() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractBlockDevices(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want []BlockDeviceMapping
	}{
		{
			"valid devices",
			[]interface{}{
				map[string]interface{}{"device_name": "/dev/xvda", "volume_id": "vol-1"},
			},
			[]BlockDeviceMapping{{DeviceName: "/dev/xvda", VolumeID: "vol-1"}},
		},
		{
			"missing fields",
			[]interface{}{
				map[string]interface{}{},
			},
			[]BlockDeviceMapping{{DeviceName: "", VolumeID: ""}},
		},
		{
			"not a list",
			map[string]interface{}{},
			[]BlockDeviceMapping{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractBlockDevices(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractBlockDevices() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseCommaList(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"normal", "a,b,c", []string{"a", "b", "c"}},
		{"spaces", "a, b , c", []string{"a", "b", "c"}},
		{"extra commas", "a,,c", []string{"a", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseCommaList(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseCommaList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToMap(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want map[string]bool
	}{
		{"normal", []string{"a", "b"}, map[string]bool{"a": true, "b": true}},
		{"empty", []string{}, map[string]bool{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToMap(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStringPointer(t *testing.T) {
	ptr := GetStringPointer("test")

	assert.NotNil(t, ptr)
	assert.Equal(t, "test", *ptr)
}
