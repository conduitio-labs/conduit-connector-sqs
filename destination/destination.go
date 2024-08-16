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

package destination

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/conduitio-labs/conduit-connector-sqs/common"
	"github.com/conduitio/conduit-commons/config"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
)

type Destination struct {
	sdk.UnimplementedDestination
	config   Config
	svc      *sqs.Client
	queueURL string

	// httpClient allows us to cleanup left over http connections. Useful to not
	// leak goroutines when tearing down the connector
	httpClient *http.Client
}

func NewDestination() sdk.Destination {
	return sdk.DestinationWithMiddleware(&Destination{
		httpClient: &http.Client{},
	}, sdk.DefaultDestinationMiddleware()...)
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

func (d *Destination) Open(ctx context.Context) (err error) {
	d.svc, err = common.NewSQSClient(ctx, d.httpClient, d.config.Config)

	if err != nil {
		return fmt.Errorf("failed to create destination sqs client: %w", err)
	}

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

const GroupIDKey = "groupID"

func (d *Destination) Write(ctx context.Context, records []opencdc.Record) (int, error) {
	for i := 0; i < len(records); i += 10 {
		end := i + 10
		if end > len(records) {
			end = len(records)
		}

		recordsChunk := records[i:end]
		sqsRecords := sqs.SendMessageBatchInput{QueueUrl: &d.queueURL}

		for _, record := range recordsChunk {
			messageBody := string(record.Bytes())

			messageAttributes := map[string]types.MessageAttributeValue{}
			for key, value := range record.Metadata {
				messageAttributes[key] = types.MessageAttributeValue{
					DataType:    aws.String("String"),
					StringValue: aws.String(value),
				}
			}

			// sqs has restrictions on what kind of characters are allowed as a batch record id.
			// As the id in this case is only relevant for the records of the batch itself, we can
			// just base64 encode the key and adapt to this limitation.
			id := base64.RawURLEncoding.EncodeToString(record.Key.Bytes())
			if len(id) > 80 {
				id = id[:80]
			}

			var messageGroupID *string
			if groupID, ok := record.Metadata[GroupIDKey]; ok {
				messageGroupID = &groupID
			}

			sqsRecords.Entries = append(sqsRecords.Entries, types.SendMessageBatchRequestEntry{
				MessageGroupId:    messageGroupID,
				MessageAttributes: messageAttributes,
				MessageBody:       &messageBody,
				DelaySeconds:      d.config.AWSSQSMessageDelay,
				Id:                &id,
			})
		}

		sendMessageOutput, err := d.svc.SendMessageBatch(ctx, &sqsRecords)
		if err != nil {
			return 0, fmt.Errorf("failed to write sqs message: %w", err)
		}

		if len(sendMessageOutput.Failed) != 0 {
			var errs []error
			for _, failed := range sendMessageOutput.Failed {
				err := fmt.Errorf("failed to deliver message (%s): %s", *failed.Id, *failed.Message)
				errs = append(errs, err)
			}

			return 0, errors.Join(errs...)
		}

		sdk.Logger(ctx).Trace().Msgf("wrote %v records", len(recordsChunk))
	}

	return len(records), nil
}

func (d *Destination) Teardown(_ context.Context) error {
	d.httpClient.CloseIdleConnections()
	return nil
}
