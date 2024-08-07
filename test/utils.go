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
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/conduitio-labs/conduit-connector-sqs/common"
	"github.com/google/uuid"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestContext(t *testing.T) context.Context {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	return logger.WithContext(context.Background())
}

func NewSQSClient(ctx context.Context, is *is.I) *sqs.Client {
	is.Helper()
	client, err := common.NewSQSClient(ctx, common.Config{
		AWSAccessKeyID:     "accessskeymock",
		AWSSecretAccessKey: "accessssecretmock",
		AWSRegion:          "us-east-1",
		AWSURL:             "http://localhost:4566",
	})
	is.NoErr(err)

	return client
}

type TestQueue struct {
	Name string

	// pointer string makes it easier to work with the aws sdk
	URL *string
}

func CreateTestQueue(ctx context.Context, t *testing.T, is *is.I, client *sqs.Client) TestQueue {
	is.Helper()
	queueName := "test-queue-" + uuid.NewString()

	_, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: &queueName,
	})
	is.NoErr(err)

	queueInput := &sqs.GetQueueUrlInput{
		QueueName: &queueName,
	}
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

func IntegrationConfig(queueName string) map[string]string {
	return map[string]string{
		common.ConfigAwsAccessKeyId:     "accessskeymock",
		common.ConfigAwsSecretAccessKey: "accessssecretmock",
		common.ConfigAwsRegion:          "us-east-1",
		common.ConfigAwsUrl:             "http://localhost:4566",
		common.ConfigAwsQueue:           queueName,
	}
}
