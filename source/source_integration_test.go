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

package source

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/conduitio-labs/conduit-connector-sqs/common"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/google/uuid"
	"github.com/matryer/is"
)

func TestSource_SuccessfulMessageReceive(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	source := NewSource()
	defer func() {
		err := source.Teardown(ctx)
		is.NoErr(err)
	}()
	sourceQueue := "test-queue-source-" + uuid.NewString()
	client, url, cfg, err := prepareIntegrationTest(t, sourceQueue)
	is.NoErr(err)

	messageBody := "Test message body"
	_, err = client.SendMessage(
		context.Background(),
		&sqs.SendMessageInput{
			MessageBody: &messageBody,
			QueueUrl:    url.QueueUrl,
		},
	)
	is.NoErr(err)

	err = source.Configure(ctx, cfg)
	is.NoErr(err)

	err = source.Open(ctx, nil)
	is.NoErr(err)

	record, err := source.Read(ctx)
	is.NoErr(err)

	err = source.Ack(ctx, record.Position)
	is.NoErr(err)

	is.Equal(string(record.Payload.After.Bytes()), messageBody)

	record, err = source.Read(ctx)
	if err != sdk.ErrBackoffRetry || record.Metadata != nil {
		t.Fatalf("expected no records and a signal that there are no more records, got %v %v", record, err)
	}

	_ = source.Teardown(ctx)
}
func TestSource_FailBadCreds(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	source := NewSource()
	defer func() {
		err := source.Teardown(ctx)
		is.NoErr(err)
	}()
	sourceQueue := "test-queue-source-" + uuid.NewString()
	_, _, cfg, err := prepareIntegrationTest(t, sourceQueue)
	is.NoErr(err)

	cfg[common.ConfigKeyAWSAccessKeyID] = ""

	err = source.Configure(ctx, cfg)
	is.NoErr(err)

	err = source.Open(ctx, nil)
	is.True(strings.Contains(err.Error(), "failed to refresh cached credentials, static credentials are empty"))
}

func TestSource_EmptyQueue(t *testing.T) {
	is := is.New(t)
	sourceQueue := "test-queue-source-" + uuid.NewString()
	_, _, cfg, err := prepareIntegrationTest(t, sourceQueue)
	ctx := context.Background()
	is.NoErr(err)

	source := NewSource()
	defer func() {
		err := source.Teardown(ctx)
		is.NoErr(err)
	}()
	err = source.Configure(ctx, cfg)
	is.NoErr(err)

	err = source.Open(ctx, nil)
	is.NoErr(err)

	record, err := source.Read(ctx)

	if err != sdk.ErrBackoffRetry || record.Metadata != nil {
		t.Fatalf("expected no records and a signal that there are no more records, got %v %v", record, err)
	}
}

func TestSource_FailEmptyQueueName(t *testing.T) {
	is := is.New(t)
	sourceQueue := ""
	_, _, _, err := prepareIntegrationTest(t, sourceQueue)

	is.True(strings.Contains(err.Error(), "Queue name cannot be empty"))
}

func prepareIntegrationTest(t *testing.T, sourceQueue string) (*sqs.Client, *sqs.GetQueueUrlOutput, map[string]string, error) {
	cfg := integrationConfig()

	client, err := newAWSClient(cfg)
	if err != nil {
		t.Fatalf("could not create S3 client: %v", err)
	}

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

	t.Cleanup(func() {
		err := deleteSQSQueue(t, client, urlResult.QueueUrl)
		if err != nil {
			t.Fatal(err)
		}
	})

	cfg[common.ConfigKeyAWSQueue] = sourceQueue

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
		config.WithRegion(cfg[common.ConfigKeyAWSRegion]),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg[common.ConfigKeyAWSAccessKeyID],
				cfg[common.ConfigKeyAWSSecretAccessKey],
				""),
		),
	)
	if err != nil {
		return nil, err
	}

	var sqsOptions []func(*sqs.Options)
	if url := cfg[common.ConfigKeyAWSURL]; url != "" {
		endpointResolver, err := common.NewEndpointResolver(url)
		if err != nil {
			return nil, err
		}

		sqsOptions = append(sqsOptions, sqs.WithEndpointResolverV2(endpointResolver))
	}

	// Create a SQS client from just a session.
	sqsClient := sqs.NewFromConfig(awsConfig, sqsOptions...)

	return sqsClient, nil
}

func integrationConfig() map[string]string {
	return map[string]string{
		common.ConfigKeyAWSAccessKeyID:     "accessskeymock",
		common.ConfigKeyAWSSecretAccessKey: "accessssecretmock",
		common.ConfigKeyAWSRegion:          "us-east-1",
	}
}
