package destination

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/google/uuid"
	"github.com/matryer/is"
)

func TestDestination_SuccessfulMessageSend(t *testing.T) {
	is := is.New(t)
	messageBody := "Test message body"

	client, url, cfg, err := prepareIntegrationTest(t)
	ctx := context.Background()
	if err != nil {
		t.Fatal(err)
	}

	if err != nil {
		t.Fatal(err)
	}

	destination := &Destination{}
	err = destination.Configure(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = destination.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}

	metadata := sdk.Metadata{}
	record := sdk.Util.Source.NewRecordCreate(
		sdk.Position("111111"),
		metadata,
		sdk.RawData("1111111"),
		sdk.RawData(messageBody),
	)

	ret, err := destination.Write(ctx, []sdk.Record{record})
	if err != nil {
		t.Fatal(err)
	}

	is.Equal(ret, 1)
	// wait a little bit after we send message, it wont show up right away
	time.Sleep(30 * time.Second)

	message, err := client.ReceiveMessage(
		context.Background(),
		&sqs.ReceiveMessageInput{
			QueueUrl: url.QueueUrl,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(message.Messages) == 0 {
		t.Fatalf("expected 1 message. Got %d", len(message.Messages))
	}

	is.Equal(*message.Messages[0].Body, messageBody)

	_ = destination.Teardown(ctx)
}

func prepareIntegrationTest(t *testing.T) (*sqs.Client, *sqs.GetQueueUrlOutput, map[string]string, error) {
	cfg, err := parseIntegrationConfig()
	if err != nil {
		t.Skip(err)
	}

	client, err := newAWSClient(cfg)
	if err != nil {
		t.Fatalf("could not create S3 client: %v", err)
	}

	sourceQueue := "test-queue-destination-" + uuid.NewString()

	_, err = client.CreateQueue(context.Background(), &sqs.CreateQueueInput{
		QueueName: &sourceQueue,
	})
	if err != nil {
		return nil, nil, nil, err
	}

	queueInput := &sqs.GetQueueUrlInput{
		QueueName: &sourceQueue,
	}
	// Get URL of queue
	urlResult, err := client.GetQueueUrl(context.Background(), queueInput)
	if err != nil {
		return nil, nil, nil, err
	}

	if err != nil {
		return nil, nil, nil, err
	}

	t.Cleanup(func() {
		err := deleteSQSQueue(t, client, urlResult.QueueUrl)
		if err != nil {
			t.Fatal(err)
		}
	})

	cfg[ConfigKeyAWSQueue] = sourceQueue

	return client, urlResult, cfg, nil
}

func deleteSQSQueue(t *testing.T, svc *sqs.Client, url *string) error {
	_, err := svc.DeleteQueue(context.Background(), &sqs.DeleteQueueInput{
		QueueUrl: url,
	})
	if err != nil {
		return err
	}
	return nil
}

func newAWSClient(cfg map[string]string) (*sqs.Client, error) {
	awsConfig, err := config.LoadDefaultConfig(context.Background(),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg[ConfigKeyAWSAccessKeyID],
				cfg[ConfigKeyAWSSecretAccessKey],
				cfg[ConfigKeyAWSToken])),
	)
	if err != nil {
		return nil, err
	}
	// Create a SQS client from just a session.
	sqsClient := sqs.NewFromConfig(awsConfig)

	return sqsClient, nil
}

func parseIntegrationConfig() (map[string]string, error) {
	awsAccessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")

	if awsAccessKeyID == "" {
		return map[string]string{}, errors.New("AWS_ACCESS_KEY_ID env var must be set")
	}

	awsSecretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if awsSecretAccessKey == "" {
		return map[string]string{}, errors.New("AWS_SECRET_ACCESS_KEY env var must be set")
	}

	awsToken := os.Getenv("AWS_TOKEN")

	return map[string]string{
		ConfigKeyAWSAccessKeyID:     awsAccessKeyID,
		ConfigKeyAWSSecretAccessKey: awsSecretAccessKey,
		ConfigKeyAWSToken:           awsToken,
	}, nil
}
