package aws

import (
	"context"
	"os"
	"testing"

	"github.com/rs/zerolog"

	"github.com/stretchr/testify/assert"
)

func TestNewEC2Service_Integration(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "fake") // optional: stub creds
	t.Setenv("AWS_SECRET_ACCESS_KEY", "fake")
	t.Setenv("AWS_REGION", "us-east-1")

	logger := zerolog.New(os.Stderr).With().Timestamp().Str("app", "drift-checker").Logger()

	service, err := NewEC2Service(context.Background(), logger)
	assert.NoError(t, err)
	assert.NotNil(t, service)
}
