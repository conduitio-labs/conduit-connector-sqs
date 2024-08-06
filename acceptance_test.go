package sqs

import (
	"fmt"
	"testing"
	"time"

	"github.com/conduitio-labs/conduit-connector-sqs/common"
	"github.com/conduitio-labs/conduit-connector-sqs/destination"
	"github.com/conduitio-labs/conduit-connector-sqs/source"
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
				// "TestDestination_Configure_Success",
				// "TestDestination_Parameters_Success",
				// "TestDestination_Write_Succes",
				// "TestSource_Configure_RequiredParams",
				// "TestSource_Configure_Success",
				// "TestSource_Open_ResumeAtPositionCDC",
				// "TestSource_Open_ResumeAtPositionSnapshot",
				// "TestSource_Parameters_Success",
				// "TestSource_Read_Success",
				// "TestSource_Read_Timeout",
				// "TestSpecifier_Exists",
				// "TestSpecifier_Specify_Success",
			},
			WriteTimeout: 5000 * time.Millisecond,
			ReadTimeout:  5000 * time.Millisecond,
		},
	})
}


func TestThings(t *testing.T) {
	is := is.New(t)
	ctx := testutils.TestContext(t)

	// testClient1, _ := testutils.NewSQSClient(ctx, is)

	// queue := testutils.CreateTestQueue(ctx, t, is, testClient1)

	// queueURL := aws.String("http://sqs.us-east-1.localhost.localstack.cloud:4566/000000000000/test-queue-3e2a8371-3435-4136-bb85-d5009417f3c2")

	// _, err := testClient1.SendMessageBatch(ctx, &sqs.SendMessageBatchInput{
	// 	Entries: []types.SendMessageBatchRequestEntry{
	// 		{
	// 			Id:          aws.String("1"),
	// 			MessageBody: aws.String("hello worlddd"),
	// 		},
	// 	},
	// 	QueueUrl: queueURL,
	// })
	// is.NoErr(err)

	// testClient2, _ := testutils.NewSQSClient(ctx, is)

	// receiveMessage := &sqs.ReceiveMessageInput{
	// 	MessageAttributeNames: []string{
	// 		string(types.QueueAttributeNameAll),
	// 	},
	// 	QueueUrl:            queueURL,
	// 	MaxNumberOfMessages: 1,
	// }

	// msgs, err := testClient2.ReceiveMessage(ctx, receiveMessage)
	// is.NoErr(err)

	// t.Log(*msgs.Messages[0].Body)

	testClient, _ := testutils.NewSQSClient(ctx, is)
	queue := testutils.CreateTestQueue(ctx, t, is, testClient)
	cfg := testutils.IntegrationConfig(queue.Name)

	source := source.NewSource()
	defer func() { is.NoErr(source.Teardown(ctx)) }()

	is.NoErr(source.Configure(ctx, cfg))
	is.NoErr(source.Open(ctx, nil))

	destination := destination.NewDestination()
	defer func() { is.NoErr(destination.Teardown(ctx)) }()

	is.NoErr(destination.Configure(ctx, cfg))
	is.NoErr(destination.Open(ctx))

	for i := 0; i < 20; i++ {
		_, err := destination.Write(ctx, []sdk.Record{
			{
				Position:  sdk.Position("1"),
				Operation: sdk.OperationSnapshot,
				Metadata:  map[string]string{},
				Key:       sdk.RawData("1"),
				Payload: sdk.Change{
					After: sdk.RawData("hello world"),
				},
			},
		})
		is.NoErr(err)
	}

	for i := 0; i < 20; i++ {
		rec, err := source.Read(ctx)
		is.NoErr(err)
		fmt.Println(string(rec.Payload.After.Bytes()))

		is.NoErr(source.Ack(ctx, rec.Position))
	}
}
