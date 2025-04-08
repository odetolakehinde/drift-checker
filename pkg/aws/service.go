package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/rs/zerolog"

	"github.com/odetolakehinde/drift-checker/pkg/common"
)

// EC2Service defines the high-level interface for interacting with EC2.
type EC2Service interface {
	GetInstance(ctx context.Context, instanceID string) (*common.EC2Instance, error)
	GetInstanceFromClient(ctx context.Context, client EC2Client, instanceID string) (*common.EC2Instance, error)
}

type ec2Service struct {
	client EC2Client
	logger zerolog.Logger
}

// NewEC2Service creates a new EC2Service facade using a configured AWS client.
func NewEC2Service(ctx context.Context, logger zerolog.Logger) (EC2Service, error) {
	log := logger.With().Str(common.LogStrLayer, "aws").Logger()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Err(err).Msg("unable to load AWS config")
		return nil, common.ErrConfigLoadFailure
	}

	client := ec2.NewFromConfig(cfg)
	return &ec2Service{
		client: client,
		logger: log,
	}, nil
}
