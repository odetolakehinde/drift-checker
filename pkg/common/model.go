package common

type (
	// EC2Instance holds the configuration details for an EC2 instance.
	EC2Instance struct {
		InstanceID          string
		InstanceType        string
		ImageID             string
		KeyName             string
		State               string
		AvailabilityZone    string
		PrivateIPAddress    string
		PublicIPAddress     string
		SubnetID            string
		VpcID               string
		SecurityGroups      []string
		Tags                map[string]string
		BlockDeviceMappings []BlockDeviceMapping
		IamInstanceProfile  string
		Monitoring          bool
		Architecture        string
		VirtualizationType  string
	}

	// BlockDeviceMapping represents the mapping of a block device.
	BlockDeviceMapping struct {
		DeviceName string
		VolumeID   string
	}

	// FieldDiff holds the values of a field that differ between AWS and Terraform.
	FieldDiff struct {
		AWS       any `json:"aws"`
		Terraform any `json:"terraform"`
	}

	// DriftResult summarizes the differences found for an EC2 instance.
	DriftResult struct {
		InstanceID    string               `json:"instance_id"`
		DriftDetected bool                 `json:"drift_detected"`
		Differences   map[string]FieldDiff `json:"differences"`
	}

	// TerraformState represents the structure of a Terraform state file.
	TerraformState struct {
		Resources []struct {
			Type      string `json:"type"`
			Name      string `json:"name"`
			Instances []struct {
				Attributes map[string]interface{} `json:"attributes"`
			} `json:"instances"`
		} `json:"resources"`
	}
)

var (
	// DefaultDriftAttributes defines the default fields checked for drift
	DefaultDriftAttributes = []string{
		"instance_type",
		"tags",
		"security_groups",
		"subnet_id",
		"image_id",
		"key_name",
		"monitoring",
		"iam_instance_profile",
		"architecture",
		"virtualization_type",
		"block_device_mappings",
	}

	// LogStrLayer is string representation of the layer level in the logs
	LogStrLayer = "layer"
	// LogStrMethod is string representation of the methods in the logs
	LogStrMethod = "method"
)
