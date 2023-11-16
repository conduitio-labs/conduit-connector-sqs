package source

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	sdk "github.com/conduitio/conduit-connector-sdk"
)

type Source struct {
	sdk.UnimplementedSource
	config   Config
	svc      *sqs.Client
	queueURL string
}

func NewSource() sdk.Source {
	return sdk.SourceWithMiddleware(&Source{}, sdk.DefaultSourceMiddleware()...)
}

func (s *Source) Parameters() map[string]sdk.Parameter {
	return Config{}.Parameters()
}

func (s *Source) Configure(ctx context.Context, cfg map[string]string) error {
	sdk.Logger(ctx).Debug().Msg("Configuring Source Connector.")

	err := sdk.Util.ParseConfig(cfg, &s.config)
	if err != nil {
		return fmt.Errorf("failed to parse source config : %w", err)
	}

	return nil
}

func (s *Source) Open(ctx context.Context, _ sdk.Position) error {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(s.config.AWSRegion),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				s.config.AWSAccessKeyID,
				s.config.AWSSecretAccessKey,
				"")),
	)
	if err != nil {
		return fmt.Errorf("failed to load aws config with given credentials : %w", err)
	}
	// Create a SQS client from just a session.
	s.svc = sqs.NewFromConfig(cfg)

	queueInput := &sqs.GetQueueUrlInput{
		QueueName: &s.config.AWSQueue,
	}
	// Get URL of queue
	urlResult, err := s.svc.GetQueueUrl(ctx, queueInput)
	if err != nil {
		return fmt.Errorf("failed to get queue amazon sqs URL: %w", err)
	}

	s.queueURL = *urlResult.QueueUrl

	return nil
}

func (s *Source) Read(ctx context.Context) (sdk.Record, error) {
	var err error
	receiveMessage := &sqs.ReceiveMessageInput{
		MessageAttributeNames: []string{
			string(types.QueueAttributeNameAll),
		},
		QueueUrl:            &s.queueURL,
		MaxNumberOfMessages: 1,
		VisibilityTimeout:   s.config.AWSSQSVisibilityTimeout,
	}

	// grab a message from queue
	sqsMessages, err := s.svc.ReceiveMessage(ctx, receiveMessage)
	if err != nil {
		return sdk.Record{}, fmt.Errorf("error retrieving amazon sqs messages: %w", err)
	}

	if len(sqsMessages.Messages) != 0 {
		attributes := sqsMessages.Messages[0].MessageAttributes
		mt := sdk.Metadata{}
		for key, value := range attributes {
			mt[key] = *value.StringValue
		}

		rec := sdk.Util.Source.NewRecordCreate(
			sdk.Position(*sqsMessages.Messages[0].ReceiptHandle),
			mt,
			sdk.RawData(*sqsMessages.Messages[0].MessageId),
			sdk.RawData(*sqsMessages.Messages[0].Body),
		)

		return rec, nil
	}

	// if there are no messages in queue, backoff
	return sdk.Record{}, sdk.ErrBackoffRetry
}

func (s *Source) Ack(ctx context.Context, position sdk.Position) error {
	// once message received in queue, remove it
	receiptHandle := string(position)
	deleteMessage := &sqs.DeleteMessageInput{
		QueueUrl:      &s.queueURL,
		ReceiptHandle: &receiptHandle,
	}

	_, err := s.svc.DeleteMessage(ctx, deleteMessage)
	if err != nil {
		return fmt.Errorf("failed to delete sqs message with receipt handle %s : %w", string(position), err)
	}

	return nil
}

func (s *Source) Teardown(_ context.Context) error {
	return nil
}
