// Package common for all common types and helper functions
package common

import (
	"fmt"
	"strings"
)

// GetString helper function to safely dereference string pointers.
func GetString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

// GetStringPointer returns a string pointer
func GetStringPointer(val string) *string {
	return &val
}

// FlattenBlockDevices converts a slice of BlockDeviceMapping structs into
// a flat list of strings in the format "device_name|volume_id".
// This representation makes it easier to compare block device mappings
// between AWS and Terraform when detecting drift.
func FlattenBlockDevices(devices []BlockDeviceMapping) []string {
	if len(devices) == 0 {
		return []string{}
	}
	flat := make([]string, 0)
	for _, d := range devices {
		flat = append(flat, d.DeviceName+"|"+d.VolumeID)
	}
	return flat
}

// ExtractBlockDevices parses block device mappings.
func ExtractBlockDevices(value interface{}) []BlockDeviceMapping {
	result := make([]BlockDeviceMapping, 0)
	if list, ok := value.([]interface{}); ok {
		for _, item := range list {
			if m, ok := item.(map[string]interface{}); ok {
				result = append(result, BlockDeviceMapping{
					DeviceName: ToString(m["device_name"]),
					VolumeID:   ToString(m["volume_id"]),
				})
			}
		}
	}

	return result
}

// ConvertToStringMap Helper to convert interface{} to map[string]string
func ConvertToStringMap(value interface{}) map[string]string {
	result := make(map[string]string)
	if rawMap, ok := value.(map[string]interface{}); ok {
		for k, v := range rawMap {
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

// ConvertToStringSlice Helper to convert interface{} to []string
func ConvertToStringSlice(value interface{}) []string {
	var result []string
	if arr, ok := value.([]interface{}); ok {
		for _, v := range arr {
			result = append(result, fmt.Sprintf("%v", v))
		}
	}
	return result
}

// ToString attempts to convert an interface{} to a string.
// If the value is not a string, it returns an empty string.
func ToString(value interface{}) string {
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

// ToBool attempts to convert an interface{} to a bool.
// If the value is not a boolean, it returns false.
func ToBool(value interface{}) bool {
	if b, ok := value.(bool); ok {
		return b
	}
	return false
}

// ParseCommaList turns a comma-separated string into a []string
func ParseCommaList(input string) []string {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	var result []string
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// ToMap converts a string slice to a lookup map
func ToMap(list []string) map[string]bool {
	m := make(map[string]bool)
	for _, item := range list {
		m[item] = true
	}

	return m
}
