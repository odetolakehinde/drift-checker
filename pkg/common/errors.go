package common

import "errors"

var (
	// ErrConfigLoadFailure indicates failure to load AWS configuration.
	ErrConfigLoadFailure = errors.New("failed to load AWS configuration")

	// ErrAWSDescribeFailure indicates a failure when calling DescribeInstances.
	ErrAWSDescribeFailure = errors.New("failed to describe EC2 instance(s)")

	// ErrInstanceNotFound indicates that the requested EC2 instance was not found in AWS.
	ErrInstanceNotFound = errors.New("instance not found in AWS")

	// ErrInvalidStateFile indicates the .tfstate file is unreadable, corrupt, or invalid.
	ErrInvalidStateFile = errors.New("invalid Terraform state file - .tfstate file is unreadable, corrupt, absent or invalid")

	// ErrTerraformInstanceMissing indicates an EC2 instance is missing in the Terraform state.
	ErrTerraformInstanceMissing = errors.New("instance not found in Terraform state")

	// ErrStateFileNotProvided indicates that no .tfstate file path was given.
	ErrStateFileNotProvided = errors.New("terraform state file path not provided")

	// ErrPromptFailed indicates a failure in collecting user input interactively.
	ErrPromptFailed = errors.New("interactive prompt failed")

	// ErrNoInstanceIDs indicates that no instance IDs were passed or entered.
	ErrNoInstanceIDs = errors.New("no EC2 instance IDs provided")
)
