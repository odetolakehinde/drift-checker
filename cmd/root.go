// Package cmd contains the CLI entry point and command-line interface logic for the application.
package cmd

import (
	"context"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/rs/zerolog"
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
	ctx := context.Background()

	// init the logger
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("app", "drift-checker").Logger()

	// init all services.
	tfSvc := tf.NewParser(ctx, logger)            // terraform service
	ec2Svc, err := aws.NewEC2Service(ctx, logger) // aws service
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize aws service")
		return
	}

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
			// check for state file. in case no state file is provided, do a fallback and ask the user
			stateFile := c.String("state-file")
			if stateFile == "" {
				stateFile, err = promptInput("Enter path to Terraform state file")
				if err != nil {
					logger.Err(err).Msg("failed to prompt input")
					return common.ErrStateFileNotProvided
				}
			}

			// check for instance IDs. in case no instance IDs are provided, do a fallback and ask the user
			instanceIDs := common.ParseCommaList(c.String("instance-ids"))
			if len(instanceIDs) == 0 {
				var raw string
				raw, err = promptInput("Enter comma-separated EC2 instance IDs")
				if err != nil {
					return common.ErrNoInstanceIDs
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
			tfInstances, err := tfSvc.Load(stateFile)
			if err != nil {
				logger.Err(err).Msg("failed to load state file")
				return err
			}

			// okay, let's get on AWS
			var awsInstances []*common.EC2Instance
			for _, id := range instanceIDs {
				inst, err := ec2Svc.GetInstance(ctx, id)
				if err != nil {
					logger.Err(err).Msgf("warning: could not retrieve AWS instance %s: %v", id, err)
					continue
				}
				awsInstances = append(awsInstances, inst)
			}

			// run all comparisons concurrently
			results := engine.CompareAllInstances(context.Background(), awsInstances, tfInstances, attributeFilter)

			// show the results
			for _, result := range results {
				engine.PrintDriftReport(result, outputJSON)
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Fatal().Err(err).Msg("failed to run app")
	}
}

// promptInput shows an interactive prompt on the CLI
func promptInput(label string) (string, error) {
	prompt := promptui.Prompt{Label: label}
	value, err := prompt.Run()
	if err != nil {
		return "", common.ErrPromptFailed
	}

	return strings.TrimSpace(value), nil
}
