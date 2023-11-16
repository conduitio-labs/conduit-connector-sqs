// Copyright © 2022 Meroxa, Inc.
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

package destination

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	sdk "github.com/conduitio/conduit-connector-sdk"
)

type Destination struct {
	sdk.UnimplementedDestination
	config   Config
	svc      *sqs.Client
	queueURL string
}

func NewDestination() sdk.Destination {
	return sdk.DestinationWithMiddleware(&Destination{}, sdk.DefaultDestinationMiddleware()...)
}

func (d *Destination) Parameters() map[string]sdk.Parameter {
	return Config{}.Parameters()
}

func (d *Destination) Configure(ctx context.Context, cfg map[string]string) error {
	sdk.Logger(ctx).Debug().Msg("Configuring Destination Connector.")

	err := sdk.Util.ParseConfig(cfg, &d.config)
	if err != nil {
		return fmt.Errorf("failed to parse destination config : %w", err)
	}

	return nil
}

func (d *Destination) Open(ctx context.Context) error {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(d.config.AWSRegion),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				d.config.AWSAccessKeyID,
				d.config.AWSSecretAccessKey,
				"")),
	)
	if err != nil {
		return fmt.Errorf("failed to load amazon config with given credentials : %w", err)
	}
	// Create a SQS client from just a session.
	d.svc = sqs.NewFromConfig(cfg)

	queueInput := &sqs.GetQueueUrlInput{
		QueueName: &d.config.AWSQueue,
	}

	// Get URL of queue
	urlResult, err := d.svc.GetQueueUrl(ctx, queueInput)
	if err != nil {
		return fmt.Errorf("failed to get sqs queue url : %w", err)
	}

	d.queueURL = *urlResult.QueueUrl

	return nil
}

func (d *Destination) Write(ctx context.Context, records []sdk.Record) (int, error) {
	for i, record := range records {
		messageBody := string(record.Payload.After.Bytes())

		messageAttributes := map[string]types.MessageAttributeValue{}
		for key, value := range record.Metadata {
			messageAttributes[key] = types.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(value),
			}
		}
		// construct record to send to destination
		sendMessageInput := &sqs.SendMessageInput{
			MessageAttributes: messageAttributes,
			MessageBody:       &messageBody,
			QueueUrl:          &d.queueURL,
			DelaySeconds:      d.config.AWSSQSMessageDelay,
		}

		_, err := d.svc.SendMessage(ctx, sendMessageInput)
		if err != nil {
			return i, fmt.Errorf("failed to write sqs message : %w", err)
		}
	}

	return len(records), nil
}

func (d *Destination) Teardown(_ context.Context) error {
	return nil // nothing to do
}
