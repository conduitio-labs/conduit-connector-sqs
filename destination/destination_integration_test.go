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
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	testutils "github.com/conduitio-labs/conduit-connector-sqs/test"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
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
	ctx := testutils.TestContext(t)

	metadata := opencdc.Metadata{}
	destination := NewDestination()
	defer func() { is.NoErr(destination.Teardown(ctx)) }()

	messageBody := "Test message body"
	record := sdk.Util.Source.NewRecordCreate(
		opencdc.Position("111111"),
		metadata,
		opencdc.RawData("1111111"),
		opencdc.RawData(messageBody),
	)

	testClient, closeTestClient := testutils.NewSQSClient(ctx, is)
	defer closeTestClient()

	testQueue := testutils.CreateTestQueue(ctx, t, is, testClient)
	cfg := testutils.IntegrationConfig(testQueue.Name)

	err := destination.Configure(ctx, cfg)
	is.NoErr(err)

	err = destination.Open(ctx)
	is.NoErr(err)

	ret, err := destination.Write(ctx, []opencdc.Record{record})
	is.NoErr(err)

	is.Equal(ret, 1)
	time.Sleep(5 * time.Second)

	message, err := testClient.ReceiveMessage(
		ctx,
		&sqs.ReceiveMessageInput{
			QueueUrl: testQueue.URL,
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
	ctx := testutils.TestContext(t)

	testClient, closeTestClient := testutils.NewSQSClient(ctx, is)
	defer closeTestClient()

	queueName := testutils.CreateTestQueue(ctx, t, is, testClient)
	cfg := testutils.IntegrationConfig(queueName.Name)

	metadata := opencdc.Metadata{}
	destination := NewDestination()
	defer func() { is.NoErr(destination.Teardown(ctx)) }()

	messageBody := "Test message body"
	record := sdk.Util.Source.NewRecordCreate(
		opencdc.Position(""),
		metadata,
		opencdc.RawData(""),
		opencdc.RawData(messageBody),
	)

	err := destination.Configure(ctx, cfg)
	is.NoErr(err)

	err = destination.Open(ctx)
	is.NoErr(err)

	_, err = destination.Write(ctx, []opencdc.Record{record})
	is.True(strings.Contains(err.Error(), "AWS.SimpleQueueService.InvalidBatchEntryId"))
}

func TestDestination_FailNonExistentQueue(t *testing.T) {
	is := is.New(t)
	ctx := testutils.TestContext(t)

	destination := NewDestination()
	defer func() { is.NoErr(destination.Teardown(ctx)) }()

	cfg := testutils.IntegrationConfig("nonexistent-testqueue")

	err := destination.Configure(ctx, cfg)
	is.NoErr(err)

	err = destination.Open(ctx)
	is.True(strings.Contains(err.Error(), "AWS.SimpleQueueService.NonExistentQueue"))
}
