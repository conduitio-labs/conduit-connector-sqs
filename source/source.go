package source

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	sdk "github.com/conduitio/conduit-connector-sdk"
	cf "github.com/meroxa/conduit-connector-amazon-sqs/config"
)

type Source struct {
	sdk.UnimplementedSource
	config      cf.Config
	svc         *sqs.Client
	queueURL    *string
	sqsMessages *sqs.ReceiveMessageOutput
}

const (
	timeout = 12 * 60 * 60
)

func NewSource() sdk.Source {
	return sdk.SourceWithMiddleware(&Source{})
}

func (s *Source) Parameters() map[string]sdk.Parameter {
	return map[string]sdk.Parameter{
		cf.ConfigKeyAWSAccessKeyID: {
			Default: "",
			Validations: []sdk.Validation{
				sdk.ValidationRequired{},
			},
			Description: "AWS Access Key ID.",
		},
		cf.ConfigKeyAWSSecretAccessKey: {
			Default: "",
			Validations: []sdk.Validation{
				sdk.ValidationRequired{},
			},
			Description: "AWS Secret Access Key.",
		},
		cf.ConfigKeyAWSToken: {
			Default:     "",
			Description: "AWS Access Token (optional).",
		},
		cf.ConfigKeyAWSQueue: {
			Default: "",
			Validations: []sdk.Validation{
				sdk.ValidationRequired{},
			},
			Description: "AWS SQS Queue Name.",
		},
	}
}

func (s *Source) Configure(ctx context.Context, cfg map[string]string) error {
	sdk.Logger(ctx).Debug().Msg("Configuring Source Connector.")
	parsedCfg, err := cf.ParseConfig(cfg)
	if err != nil {
		return err
	}
	s.config = parsedCfg

	return nil
}

func (s *Source) Open(ctx context.Context, _ sdk.Position) error {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				s.config.AWSAccessKeyID,
				s.config.AWSSecretAccessKey,
				s.config.AWSToken)),
	)
	if err != nil {
		return err
	}
	// Create a SQS client from just a session.
	s.svc = sqs.NewFromConfig(cfg)

	queueInput := &sqs.GetQueueUrlInput{
		QueueName: &s.config.AWSQueue,
	}
	// Get URL of queue
	urlResult, err := s.svc.GetQueueUrl(ctx, queueInput)
	if err != nil {
		return err
	}

	s.queueURL = urlResult.QueueUrl

	return nil
}

func (s *Source) Read(ctx context.Context) (sdk.Record, error) {
	var err error
	receiveMessage := &sqs.ReceiveMessageInput{
		MessageAttributeNames: []string{
			string(types.QueueAttributeNameAll),
		},
		QueueUrl:            s.queueURL,
		MaxNumberOfMessages: 1,
		VisibilityTimeout:   timeout,
	}

	// grab a message from queue
	s.sqsMessages, err = s.svc.ReceiveMessage(ctx, receiveMessage)
	if err != nil {
		return sdk.Record{}, err
	}

	if s.sqsMessages.Messages != nil {
		attributes := s.sqsMessages.Messages[0].MessageAttributes
		mt := sdk.Metadata{}
		for key, value := range attributes {
			mt[key] = *value.StringValue
		}

		rec := sdk.Util.Source.NewRecordCreate(
			sdk.Position(*s.sqsMessages.Messages[0].ReceiptHandle),
			mt,
			sdk.RawData(*s.sqsMessages.Messages[0].MessageId),
			sdk.RawData(*s.sqsMessages.Messages[0].Body),
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
		QueueUrl:      s.queueURL,
		ReceiptHandle: &receiptHandle,
	}

	_, err := s.svc.DeleteMessage(ctx, deleteMessage)
	if err != nil {
		return err
	}

	return nil
}

func (s *Source) Teardown(_ context.Context) error {
	return nil
}
