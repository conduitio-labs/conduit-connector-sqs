package sqs

import (
	"testing"
	"time"

	"github.com/conduitio-labs/conduit-connector-sqs/common"
	testutils "github.com/conduitio-labs/conduit-connector-sqs/test"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/matryer/is"
	"go.uber.org/goleak"
)

func TestAcceptance(t *testing.T) {
	is := is.New(t)

	ctx := testutils.TestContext(t)
	sourceConfig := testutils.IntegrationConfig("")
	destinationConfig := testutils.IntegrationConfig("")

	sdk.AcceptanceTest(t, sdk.ConfigurableAcceptanceTestDriver{
		Config: sdk.ConfigurableAcceptanceTestDriverConfig{
			Context:           ctx,
			GoleakOptions:     []goleak.Option{goleak.IgnoreCurrent()},
			Connector:         Connector,
			SourceConfig:      sourceConfig,
			DestinationConfig: destinationConfig,
			BeforeTest: func(t *testing.T) {
				// sqs client creation must be created and cleaned up inside
				// BeforeTest so that goleak doesn't alert of a false positive http
				// connection leaking.
				testClient, close := testutils.NewSQSClient(ctx, is)
				defer close()

				queue := testutils.CreateTestQueue(ctx, t, is, testClient)
				sourceConfig[common.ConfigKeyAWSQueue] = queue.Name
				destinationConfig[common.ConfigKeyAWSQueue] = queue.Name

				sdk.Logger(ctx).Info().Msgf("queue name: %v", queue.Name)
				sdk.Logger(ctx).Info().Msgf("queue url: %v", *queue.URL)
			},
			Skip: []string{
				"TestSource_Configure_RequiredParams",
				"TestDestination_Configure_RequiredParams",
				"TestDestination_Configure_Success",
				"TestDestination_Parameters_Success",
				// "TestDestination_Write_Success",
				"TestSource_Configure_RequiredParams",
				"TestSource_Configure_Success",
				"TestSource_Open_ResumeAtPositionCDC",
				"TestSource_Open_ResumeAtPositionSnapshot",
				"TestSource_Parameters_Success",
				"TestSource_Read_Success",
				"TestSource_Read_Timeout",
				"TestSpecifier_Exists",
				"TestSpecifier_Specify_Success",
			},
			WriteTimeout: 5000 * time.Millisecond,
			ReadTimeout:  5000 * time.Millisecond,
		},
	})
}
