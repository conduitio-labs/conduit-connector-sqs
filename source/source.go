// Copyright © 2024 Meroxa, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package source

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/conduitio-labs/conduit-connector-sqs/common"
	"github.com/conduitio/conduit-commons/lang"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
)

type Source struct {
	sdk.UnimplementedSource
	config        Config
	svc           *sqs.Client
	queueURL      string
	savedMessages []types.Message

	// receiveMessageCalled will be called each time the `ReceiveMessage` method
	// from the SQS client is called. This is useful in tests only; in non-test
	// environments, calling this should have no side effects.
	receiveMessageCalled func()

	// httpClient allows us to cleanup left over http connections. Useful to not
	// leak goroutines when tearing down the connector
	httpClient *http.Client
}

func (s *Source) Config() sdk.SourceConfig {
	return &s.config
}

// newSource initializes a source without any middlewares. Useful for integration test setup.
func newSource() *Source {
	return &Source{
		config: Config{
			DefaultSourceMiddleware: sdk.DefaultSourceMiddleware{
				SourceWithSchemaExtraction: sdk.SourceWithSchemaExtraction{
					PayloadEnabled: lang.Ptr(false),
					KeyEnabled:     lang.Ptr(false),
				},
			},
		},

		httpClient: &http.Client{},
	}
}

func NewSource() sdk.Source {
	return sdk.SourceWithMiddleware(newSource())
}

func (s *Source) Open(ctx context.Context, sdkPos opencdc.Position) (err error) {
	s.svc, err = common.NewSQSClient(ctx, s.httpClient, s.config.Config)
	if err != nil {
		return fmt.Errorf("failed to create source sqs client: %w", err)
	}

	sdk.Logger(ctx).Info().Msg("connected to sqs")

	queueName := &s.config.QueueName
	if sdkPos != nil {
		pos, err := common.ParsePosition(sdkPos)
		if err != nil {
			return fmt.Errorf("failed to parse source position: %w", err)
		}

		if s.config.QueueName != "" && s.config.QueueName != pos.QueueName {
			return fmt.Errorf(
				"the old position contains a different queue name than the connector configuration (%q vs %q), please check if the configured queue name changed since the last run",
				pos.QueueName, s.config.QueueName,
			)
		}

		sdk.Logger(ctx).Debug().Msg("queue name from position matches configured queue")
	}

	queueInput := &sqs.GetQueueUrlInput{QueueName: queueName}
	urlResult, err := s.svc.GetQueueUrl(ctx, queueInput)
	if err != nil {
		return fmt.Errorf("failed to get queue amazon sqs URL: %w", err)
	}

	s.queueURL = *urlResult.QueueUrl

	sdk.Logger(ctx).Info().Msgf("listening to queue %v", s.queueURL)

	return nil
}

func (s *Source) receiveMessage(ctx context.Context) (msg types.Message, err error) {
	if len(s.savedMessages) >= 1 {
		first := s.savedMessages[0]
		s.savedMessages = s.savedMessages[1:]
		return first, nil
	}

	receiveMessage := &sqs.ReceiveMessageInput{
		MessageAttributeNames: []string{
			string(types.QueueAttributeNameAll),
		},
		QueueUrl:            &s.queueURL,
		MaxNumberOfMessages: s.config.MaxNumberOfMessages,
		VisibilityTimeout:   s.config.VisibilityTimeout,
		WaitTimeSeconds:     s.config.WaitTimeSeconds,
	}

	sqsMessages, err := s.svc.ReceiveMessage(ctx, receiveMessage)
	if err != nil {
		return msg, fmt.Errorf("error retrieving amazon sqs messages: %w", err)
	}
	if s.receiveMessageCalled != nil {
		s.receiveMessageCalled()
	}

	if len(sqsMessages.Messages) == 0 {
		sdk.Logger(ctx).Warn().Msg("got 0 messages from queue")
		return msg, sdk.ErrBackoffRetry
	}

	msg = sqsMessages.Messages[0]
	if len(sqsMessages.Messages) == 1 {
		return msg, nil
	}

	s.savedMessages = sqsMessages.Messages[1:]
	return msg, nil
}

func (s *Source) Read(ctx context.Context) (rec opencdc.Record, err error) {
	message, err := s.receiveMessage(ctx)
	if err != nil {
		return rec, err
	}

	mt := opencdc.Metadata{}
	for key, value := range message.MessageAttributes {
		mt[key] = *value.StringValue
	}

	position := common.Position{
		ReceiptHandle: *message.ReceiptHandle,
		QueueName:     s.config.QueueName,
	}.ToSdkPosition()

	rec = sdk.Util.Source.NewRecordCreate(
		position, mt,
		opencdc.RawData(*message.MessageId),
		opencdc.RawData(*message.Body),
	)
	return rec, nil
}

func (s *Source) Ack(ctx context.Context, sdkPos opencdc.Position) error {
	position, err := common.ParsePosition(sdkPos)
	if err != nil {
		return fmt.Errorf("failed to parse position: %w", err)
	}

	deleteMessage := &sqs.DeleteMessageInput{
		QueueUrl:      &s.queueURL,
		ReceiptHandle: &position.ReceiptHandle,
	}

	if _, err := s.svc.DeleteMessage(ctx, deleteMessage); err != nil {
		return fmt.Errorf(
			"failed to delete sqs message with receipt handle %s: %w",
			position.ReceiptHandle, err)
	}

	return nil
}

func (s *Source) Teardown(_ context.Context) error {
	s.httpClient.CloseIdleConnections()
	return nil
}
