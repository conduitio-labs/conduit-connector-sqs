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
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/conduitio-labs/conduit-connector-sqs/common"
	"github.com/conduitio/conduit-commons/config"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
	cmap "github.com/orcaman/concurrent-map/v2"
)

type Destination struct {
	sdk.UnimplementedDestination
	config          Config
	svc             *sqs.Client
	queueNameParser queueNameParser
	queueURLMap     *queueURLMap

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

	switch queue := d.config.AWSQueue; {
	case isGoTemplate(queue):
		parser, err := newFromGoTemplateParser(queue)
		if err != nil {
			return fmt.Errorf("failed to create template parser: %w", err)
		}
		d.queueNameParser = parser
	default:
		d.queueNameParser = fromCollectionParser{queue}
	}

	return nil
}

func (d *Destination) Open(ctx context.Context) (err error) {
	d.svc, err = common.NewSQSClient(ctx, d.httpClient, d.config.Config)
	if err != nil {
		return fmt.Errorf("failed to create destination sqs client: %w", err)
	}

	d.queueURLMap = newQueueURLMap(d.svc)

	if !isGoTemplate(d.config.AWSQueue) {
		// we can be nice to the user and inform them that the queue that they did
		// specify exists and that the connector has access to it.

		urlResult, err := d.svc.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
			QueueName: &d.config.AWSQueue,
		})
		if err != nil {
			return fmt.Errorf("failed to get sqs queue url : %w", err)
		}

		queueURL := *urlResult.QueueUrl

		sdk.Logger(ctx).Info().Msgf("writing to queue %v", queueURL)
	}

	return nil
}

const (
	GroupIDKey = "groupID"
	DedupIDKey = "deduplicationID"
)

func (d *Destination) Write(ctx context.Context, records []opencdc.Record) (int, error) {
	batches, err := d.splitIntoBatches(records)
	if err != nil {
		return 0, fmt.Errorf("failed to create batches: %w", err)
	}

	return d.writeBatches(ctx, batches)
}

func (d *Destination) Teardown(_ context.Context) error {
	d.httpClient.CloseIdleConnections()
	return nil
}

func isGoTemplate(template string) bool {
	return strings.HasPrefix(template, "{{") && strings.HasSuffix(template, "}}")
}

type queueURLMap struct {
	client *sqs.Client
	cache  cmap.ConcurrentMap[string, string]
}

func newQueueURLMap(client *sqs.Client) *queueURLMap {
	cache := cmap.New[string]()
	return &queueURLMap{
		client: client,
		cache:  cache,
	}
}

func (m *queueURLMap) getURLForQueue(ctx context.Context, queueName string) (string, error) {
	if url, ok := m.cache.Get(queueName); ok {
		return url, nil
	}

	queueURLOutput, err := m.client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to fetch queue url to cache: %w", err)
	}

	url := *queueURLOutput.QueueUrl

	m.cache.Set(queueName, url)

	return url, nil
}

type queueNameParser interface {
	ParseQueueName(opencdc.Record) (string, error)
}

// fromCollectionParser parses queue names from the record collection field. If
// none found, it will return the defaultQueueName as the obtained queue name.
type fromCollectionParser struct {
	defaultQueueName string
}

func (parser fromCollectionParser) ParseQueueName(rec opencdc.Record) (string, error) {
	queueName, err := rec.Metadata.GetCollection()
	if err != nil {
		return parser.defaultQueueName, nil
	}

	return queueName, nil
}

// fromGoTemplateParser parses queue names from the given go template.
type fromGoTemplateParser struct {
	contents string
	template *template.Template
}

func newFromGoTemplateParser(templateContents string) (*fromGoTemplateParser, error) {
	t, err := template.New("parser").Parse(templateContents)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &fromGoTemplateParser{templateContents, t}, nil
}

func (parser fromGoTemplateParser) ParseQueueName(rec opencdc.Record) (string, error) {
	var sb strings.Builder
	if err := parser.template.Execute(&sb, rec); err != nil {
		return "", fmt.Errorf("failed to parse streamName from template: %w", err)
	}

	if sb.Len() == 0 {
		return "", fmt.Errorf(
			"streamName not found in record %s from template %s",
			string(rec.Key.Bytes()), parser.contents,
		)
	}

	return sb.String(), nil
}

type messageBatch struct {
	queueName string
	records   []opencdc.Record
}

func (d *Destination) splitIntoBatches(recs []opencdc.Record) ([]messageBatch, error) {
	var batches []messageBatch
	for _, rec := range recs {
		queueName, err := d.queueNameParser.ParseQueueName(rec)
		if err != nil {
			return nil, fmt.Errorf("failed to parse queue name while batching: %w", err)
		}

		if len(batches) == 0 {
			batches = append(batches, messageBatch{
				queueName: queueName,
				records:   []opencdc.Record{rec},
			})
			continue
		}

		if batchRef := &batches[len(batches)-1]; batchRef.queueName == queueName {
			batchRef.records = append(batchRef.records, rec)
		} else {
			batches = append(batches, messageBatch{
				queueName: queueName,
				records:   []opencdc.Record{rec},
			})
		}
	}

	return batches, nil
}

func (d *Destination) writeBatches(ctx context.Context, batches []messageBatch) (int, error) {
	var written int
	for _, batch := range batches {
		records := batch.records
		for i := 0; i < len(records); i += 10 {
			end := i + 10
			if end > len(records) {
				end = len(records)
			}

			recordsChunk := records[i:end]

			queueURL, err := d.queueURLMap.getURLForQueue(ctx, batch.queueName)
			if err != nil {
				return 0, fmt.Errorf("failed to get queue url while writing batch: %w", err)
			}

			sqsRecords := sqs.SendMessageBatchInput{QueueUrl: &queueURL}

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

				var messageGroupID string
				if groupID, ok := record.Metadata[GroupIDKey]; ok {
					messageGroupID = groupID
				}

				sqsRecords.Entries = append(sqsRecords.Entries, types.SendMessageBatchRequestEntry{
					MessageGroupId:    &messageGroupID,
					MessageAttributes: messageAttributes,
					MessageBody:       &messageBody,
					DelaySeconds:      d.config.MessageDelay,
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
			written += len(recordsChunk)
		}
	}

	return written, nil
}
