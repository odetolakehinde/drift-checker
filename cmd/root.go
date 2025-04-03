// Package cmd contains the CLI entry point and command-line interface logic for the application.
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"

	"github.com/odetolakehinde/drift-checker/pkg/aws"
	"github.com/odetolakehinde/drift-checker/pkg/common"
	"github.com/odetolakehinde/drift-checker/pkg/engine"
	tf "github.com/odetolakehinde/drift-checker/pkg/terraform"
)

// Run initializes and executes the command-line interface for the tool/application.
//
// It defines CLI flags, handles user input (with interactive prompts if flags are missing),
// loads EC2 and Terraform data, and invokes the drift detection engine.
func Run() {
	app := &cli.App{
		Name:  "drift-checker",
		Usage: "Detect drift between AWS EC2 instances and Terraform state",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "state-file", Usage: "Path to Terraform .tfstate file"},
			&cli.StringFlag{Name: "instance-ids", Usage: "Comma-separated list of EC2 instance IDs"},
			&cli.StringFlag{Name: "attributes", Usage: "Comma-separated attributes to check for drift"},
			&cli.BoolFlag{Name: "json", Usage: "Output drift result as JSON"},
		},
		Action: func(c *cli.Context) error {
			ctx := context.Background()

			// check for state file. in case no state file is provided, do a fallback and ask the user
			stateFile := c.String("state-file")
			if stateFile == "" {
				var err error
				stateFile, err = promptInput("Enter path to Terraform state file")
				if err != nil {
					return err
				}
			}

			// check for instance IDs. in case no instance IDs are provided, do a fallback and ask the user
			instanceIDs := common.ParseCommaList(c.String("instance-ids"))
			if len(instanceIDs) == 0 {
				raw, err := promptInput("Enter comma-separated EC2 instance IDs")
				if err != nil {
					return err
				}
				instanceIDs = common.ParseCommaList(raw)
			}

			// we need to fetch the attributes we want to compare
			attrInput := c.String("attributes")
			if attrInput == "" {
				// if nothing, use the default attributes we've already defined.
				attrInput = strings.Join(common.DefaultDriftAttributes, ",")
			}
			attributeFilter := common.ToMap(common.ParseCommaList(attrInput))

			outputJSON := c.Bool("json")

			// time to parse the Terraform file
			tfInstances, err := tf.ParseTerraformState(stateFile)
			if err != nil {
				return fmt.Errorf("failed to parse Terraform state: %w", err)
			}

			// okay, let's get on AWS
			var awsInstances []*common.EC2Instance
			for _, id := range instanceIDs {
				inst, err := aws.GetInstance(ctx, id)
				if err != nil {
					log.Printf("warning: could not retrieve AWS instance %s: %v", id, err)
					continue
				}
				awsInstances = append(awsInstances, inst)
			}

			// run all comparisons concurrently
			results := engine.CompareAllInstances(awsInstances, tfInstances, attributeFilter)

			// show the results
			for _, result := range results {
				engine.PrintDriftReport(result, outputJSON)
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "❌", err)
		os.Exit(1)
	}
}

// promptInput shows an interactive prompt on the CLI
func promptInput(label string) (string, error) {
	prompt := promptui.Prompt{Label: label}
	value, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("prompt error: %w", err)
	}
	return strings.TrimSpace(value), nil
}
