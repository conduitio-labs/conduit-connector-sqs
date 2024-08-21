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

package testutils

import (
	"context"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/conduitio-labs/conduit-connector-sqs/common"
	"github.com/conduitio/conduit-commons/config"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/google/uuid"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestContext(t *testing.T) context.Context {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	return logger.WithContext(context.Background())
}

func NewSQSClient(ctx context.Context, is *is.I) (*sqs.Client, func()) {
	is.Helper()

	httpClient := &http.Client{}

	client, err := common.NewSQSClient(ctx, httpClient, common.Config{
		AWSAccessKeyID:     "accessskeymock",
		AWSSecretAccessKey: "accessssecretmock",
		AWSRegion:          "us-east-1",
		AWSURL:             "http://localhost:4566",
	})
	is.NoErr(err)

	return client, func() {
		httpClient.CloseIdleConnections()
	}
}

type TestQueue struct {
	Name string

	// pointer string makes it easier to work with the aws sdk
	URL *string
}

func CreateTestQueue(ctx context.Context, t *testing.T, is *is.I, client *sqs.Client) TestQueue {
	is.Helper()
	queueName := "test-queue-" + uuid.NewString()[:8]

	_, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{QueueName: &queueName})
	is.NoErr(err)

	queueInput := &sqs.GetQueueUrlInput{QueueName: &queueName}
	urlResult, err := client.GetQueueUrl(ctx, queueInput)
	is.NoErr(err)

	t.Cleanup(func() {
		_, err := client.DeleteQueue(ctx, &sqs.DeleteQueueInput{
			QueueUrl: urlResult.QueueUrl,
		})
		is.NoErr(err)
	})

	return TestQueue{
		Name: queueName,
		URL:  urlResult.QueueUrl,
	}
}

func CreateTestFifoQueue(ctx context.Context, t *testing.T, is *is.I, client *sqs.Client) TestQueue {
	is.Helper()
	queueName := "test-queue-" + uuid.NewString()[:8] + ".fifo"

	_, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: &queueName,
		Attributes: map[string]string{
			"FifoQueue":                 "true",
			"ContentBasedDeduplication": "true",
		},
	})
	is.NoErr(err)

	queueInput := &sqs.GetQueueUrlInput{QueueName: &queueName}
	urlResult, err := client.GetQueueUrl(ctx, queueInput)
	is.NoErr(err)

	t.Cleanup(func() {
		_, err := client.DeleteQueue(ctx, &sqs.DeleteQueueInput{
			QueueUrl: urlResult.QueueUrl,
		})
		is.NoErr(err)
	})

	return TestQueue{
		Name: queueName,
		URL:  urlResult.QueueUrl,
	}
}

func SourceConfig(queueName string) config.Config {
	return config.Config{
		"aws.accessKeyId":       "accessskeymock",
		"aws.secretAccessKey":   "accessssecretmock",
		"aws.region":            "us-east-1",
		"aws.url":               "http://localhost:4566",
		"aws.queue":             queueName,
		"aws.visibilityTimeout": "1",
		"aws.waitTimeSeconds":   "0",
	}
}

func DestinationConfig(queueName string) config.Config {
	return config.Config{
		"aws.accessKeyId":     "accessskeymock",
		"aws.secretAccessKey": "accessssecretmock",
		"aws.region":          "us-east-1",
		"aws.url":             "http://localhost:4566",
		"aws.queue":           queueName,
	}
}

func StartSource(ctx context.Context, is *is.I, s sdk.Source, queueName string) (sdk.Source, func()) {
	is.NoErr(s.Configure(ctx, SourceConfig(queueName)))
	is.NoErr(s.Open(ctx, nil))

	return s, func() {
		is.NoErr(s.Teardown(ctx))
	}
}

func StartDestination(
	ctx context.Context,
	is *is.I,
	d sdk.Destination,
	queueName string,
) (sdk.Destination, func()) {
	is.NoErr(d.Configure(ctx, SourceConfig(queueName)))
	is.NoErr(d.Open(ctx))

	return d, func() {
		is.NoErr(d.Teardown(ctx))
	}
}
