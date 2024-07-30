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
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/conduitio-labs/conduit-connector-sqs/common"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/google/uuid"
	"github.com/matryer/is"
)

type ResultConfig struct {
	Payload   Data   `json:"payload"`
	MetaData  string `json:"meta_data"`
	Position  string `json:"position"`
	Key       string `json:"key"`
	Operation string `json:"operation"`
}

type Data struct {
	Before string `json:"before"`
	After  string `json:"after"`
}

func TestDestination_SuccessfulMessageSend(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	metadata := sdk.Metadata{}
	destination := NewDestination()
	defer func() {
		err := destination.Teardown(ctx)
		is.NoErr(err)
	}()

	messageBody := "Test message body"
	record := sdk.Util.Source.NewRecordCreate(
		sdk.Position("111111"),
		metadata,
		sdk.RawData("1111111"),
		sdk.RawData(messageBody),
	)

	client, url, cfg, err := prepareIntegrationTest(t)
	is.NoErr(err)

	err = destination.Configure(ctx, cfg)
	is.NoErr(err)

	err = destination.Open(ctx)
	is.NoErr(err)

	ret, err := destination.Write(ctx, []sdk.Record{record})
	is.NoErr(err)

	is.Equal(ret, 1)
	time.Sleep(5 * time.Second)

	message, err := client.ReceiveMessage(
		context.Background(),
		&sqs.ReceiveMessageInput{
			QueueUrl: url.QueueUrl,
		},
	)

	is.NoErr(err)
	is.Equal(len(message.Messages), 1)

	var result ResultConfig
	err = json.Unmarshal([]byte(*message.Messages[0].Body), &result)
	is.NoErr(err)
	bodyDecoded, err := base64.StdEncoding.DecodeString(result.Payload.After)

	is.NoErr(err)
	is.Equal(string(bodyDecoded), messageBody)
}

func TestDestination_FailBadRecord(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	metadata := sdk.Metadata{}
	destination := NewDestination()
	defer func() {
		err := destination.Teardown(ctx)
		is.NoErr(err)
	}()

	messageBody := "Test message body"
	record := sdk.Util.Source.NewRecordCreate(
		sdk.Position(""),
		metadata,
		sdk.RawData(""),
		sdk.RawData(messageBody),
	)

	_, _, cfg, err := prepareIntegrationTest(t)
	is.NoErr(err)

	err = destination.Configure(ctx, cfg)
	is.NoErr(err)

	err = destination.Open(ctx)
	is.NoErr(err)

	_, err = destination.Write(ctx, []sdk.Record{record})
	is.True(strings.Contains(err.Error(), "AWS.SimpleQueueService.InvalidBatchEntryId"))
}

func TestDestination_FailNonExistentQueue(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	destination := NewDestination()
	defer func() {
		err := destination.Teardown(ctx)
		is.NoErr(err)
	}()

	_, _, cfg, err := prepareIntegrationTest(t)
	is.NoErr(err)

	cfg[common.ConfigKeyAWSQueue] = ""

	err = destination.Configure(ctx, cfg)
	is.NoErr(err)

	err = destination.Open(ctx)
	is.True(strings.Contains(err.Error(), "AWS.SimpleQueueService.NonExistentQueue"))
}

func prepareIntegrationTest(t *testing.T) (*sqs.Client, *sqs.GetQueueUrlOutput, map[string]string, error) {
	cfg := integrationConfig()

	sourceQueue := "test-queue-destination-" + uuid.NewString()

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
				cfg[common.ConfigKeyAWSSecretAccessKey], "")),
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
