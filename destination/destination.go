// Copyright © 2023 Meroxa, Inc.
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
	configv2 "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/conduitio/conduit-commons/config"
	"github.com/conduitio/conduit-commons/opencdc"
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

func (d *Destination) Parameters() config.Parameters {
	return Config{}.Parameters()
}

func (d *Destination) Configure(ctx context.Context, cfg config.Config) error {
	sdk.Logger(ctx).Debug().Msg("Configuring Destination Connector.")

	err := sdk.Util.ParseConfig(ctx, cfg, &d.config, NewDestination().Parameters())
	if err != nil {
		return fmt.Errorf("failed to parse destination config : %w", err)
	}

	return nil
}

func (d *Destination) Open(ctx context.Context) error {
	cfg, err := configv2.LoadDefaultConfig(ctx,
		configv2.WithRegion(d.config.AWSRegion),
		configv2.WithCredentialsProvider(
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

func (d *Destination) Write(ctx context.Context, records []opencdc.Record) (int, error) {
	for i := 0; i < len(records); i += 10 {
		end := i + 10
		if end > len(records) {
			end = len(records)
		}

		recordsChunk := records[i:end]
		sqsRecords := sqs.SendMessageBatchInput{
			QueueUrl: &d.queueURL,
		}

		for _, record := range recordsChunk {
			messageBody := string(record.Bytes())

			messageAttributes := map[string]types.MessageAttributeValue{}
			for key, value := range record.Metadata {
				messageAttributes[key] = types.MessageAttributeValue{
					DataType:    aws.String("String"),
					StringValue: aws.String(value),
				}
			}
			id := string(record.Key.Bytes())
			// construct record to send to destination
			sqsRecords.Entries = append(sqsRecords.Entries, types.SendMessageBatchRequestEntry{
				MessageAttributes: messageAttributes,
				MessageBody:       &messageBody,
				DelaySeconds:      d.config.AWSSQSMessageDelay,
				Id:                &id,
			})
		}

		_, err := d.svc.SendMessageBatch(ctx, &sqsRecords)
		if err != nil {
			return 0, fmt.Errorf("failed to write sqs message : %w", err)
		}
	}

	return len(records), nil
}

func (d *Destination) Teardown(_ context.Context) error {
	return nil // nothing to do
}
