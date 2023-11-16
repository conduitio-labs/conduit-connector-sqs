package source

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/google/uuid"
	"github.com/matryer/is"
)

func TestSource_SuccessfulMessageReceive(t *testing.T) {
	is := is.New(t)

	client, url, cfg, err := prepareIntegrationTest(t)
	ctx := context.Background()
	if err != nil {
		t.Fatal(err)
	}
	messageBody := "Test message body"

	// Send a message to the queue

	_, err = client.SendMessage(
		context.Background(),
		&sqs.SendMessageInput{
			MessageBody: &messageBody,
			QueueUrl:    url.QueueUrl,
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	source := &Source{}
	err = source.Configure(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = source.Open(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	record, err := source.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}

	is.Equal(string(record.Payload.After.Bytes()), messageBody)

	_ = source.Teardown(ctx)
}

func TestSource_EmptyQueue(t *testing.T) {
	_, _, cfg, err := prepareIntegrationTest(t)
	ctx := context.Background()
	if err != nil {
		t.Fatal(err)
	}

	source := &Source{}
	err = source.Configure(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = source.Open(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	record, err := source.Read(ctx)

	if err != sdk.ErrBackoffRetry || record.Metadata != nil {
		t.Fatalf("expected no records and a signal that there are no more records, got %v %v", record, err)
	}
	_ = source.Teardown(ctx)
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

	sourceQueue := "test-queue-source-" + uuid.NewString()

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
		config.WithRegion(cfg[ConfigKeyAWSRegion]),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg[ConfigKeyAWSAccessKeyID],
				cfg[ConfigKeyAWSSecretAccessKey],
				"")),
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

	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		return map[string]string{}, errors.New("AWS_REGION env var must be set")
	}

	awsVisibility := os.Getenv("AWS_VISIBILITY")

	return map[string]string{
		ConfigKeyAWSAccessKeyID:       awsAccessKeyID,
		ConfigKeyAWSSecretAccessKey:   awsSecretAccessKey,
		ConfigKeySQSVisibilityTimeout: awsVisibility,
		ConfigKeyAWSRegion:            awsRegion,
	}, nil
}
