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
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/conduitio-labs/conduit-connector-sqs/common"
	testutils "github.com/conduitio-labs/conduit-connector-sqs/test"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/matryer/is"
)

func sendMessage(ctx context.Context, is *is.I, client *sqs.Client, queueURL, msg *string) {
	is.Helper()
	_, err := client.SendMessage(
		ctx,
		&sqs.SendMessageInput{
			MessageBody: msg,
			QueueUrl:    queueURL,
		},
	)
	is.NoErr(err)
}

func TestSource_SuccessfulMessageReceive(t *testing.T) {
	is := is.New(t)
	ctx := testutils.TestContext(t)

	testClient, cleanTestClient := testutils.NewSQSClient(ctx, is)
	defer cleanTestClient()

	testQueue := testutils.CreateTestQueue(ctx, t, is, testClient)

	source, cleanSource := testutils.StartSource(ctx, is, NewSource(), testQueue.Name)
	defer cleanSource()

	messageBody := "Test message body"
	sendMessage(ctx, is, testClient, testQueue.URL, &messageBody)

	record, err := source.Read(ctx)
	is.NoErr(err)

	err = source.Ack(ctx, record.Position)
	is.NoErr(err)

	is.Equal(string(record.Payload.After.Bytes()), messageBody)

	record, err = source.Read(ctx)
	if err != sdk.ErrBackoffRetry || record.Metadata != nil {
		t.Fatalf("expected no records and a signal that there are no more records, got %v %v", record, err)
	}
}

func TestSource_OpenWithPosition(t *testing.T) {
	is := is.New(t)
	ctx := testutils.TestContext(t)

	testClient, cleanTestClient := testutils.NewSQSClient(ctx, is)
	defer cleanTestClient()

	testQueue := testutils.CreateTestQueue(ctx, t, is, testClient)
	cfg := testutils.SourceConfig(testQueue.Name)
	{
		source := NewSource()
		is.NoErr(source.Configure(ctx, cfg))

		pos := common.Position{
			ReceiptHandle: "test-handle",
			QueueName:     testQueue.Name,
		}

		is.NoErr(source.Open(ctx, pos.ToSdkPosition()))
	}
	{
		source := NewSource()
		is.NoErr(source.Configure(ctx, cfg))

		pos := common.Position{
			ReceiptHandle: "test-handle",
			QueueName:     "other-test-queue",
		}

		err := source.Open(ctx, pos.ToSdkPosition())
		if err == nil {
			is.Fail() // expected error on wrong position
		}

		is.True(strings.Contains(
			err.Error(),
			"the old position contains a different queue name than the connector configuration",
		))
	}
}

func TestMultipleMessageFetch(t *testing.T) {
	is := is.New(t)
	ctx := testutils.TestContext(t)

	testClient, cleanTestClient := testutils.NewSQSClient(ctx, is)
	defer cleanTestClient()

	testQueue := testutils.CreateTestQueue(ctx, t, is, testClient)

	totalMessages := 20
	maxNumberOfMessages := 5

	expectedMessages := make([]string, totalMessages)
	for i := range totalMessages {
		msg := fmt.Sprintf("message %d", i)
		sendMessage(ctx, is, testClient, testQueue.URL, &msg)
		expectedMessages[i] = msg
	}

	source := newSource()
	var receiveMessageCalls int

	source.receiveMessageCalled = func() {
		receiveMessageCalls++
	}

	cfg := testutils.SourceConfig(testQueue.Name)
	cfg[ConfigAwsVisibilityTimeout] = "10"
	cfg[ConfigAwsMaxNumberOfMessages] = fmt.Sprint(maxNumberOfMessages)

	is.NoErr(source.Configure(ctx, cfg))
	is.NoErr(source.Open(ctx, nil))
	defer func() { is.NoErr(source.Teardown(ctx)) }()

	recs := make([]opencdc.Record, totalMessages)
	for i := range totalMessages {
		rec, err := source.Read(ctx)
		is.NoErr(err)
		is.NoErr(source.Ack(ctx, rec.Position))
		recs[i] = rec
	}

	// records might come unsorted
	sort.Slice(recs, func(i, j int) bool {
		prevInt, _ := strconv.Atoi(string(recs[i].Payload.After.Bytes())[len("message "):])
		nextInt, _ := strconv.Atoi(string(recs[j].Payload.After.Bytes())[len("message "):])
		return prevInt < nextInt
	})

	// assert record contents
	for i := range recs {
		expected := expectedMessages[i]
		actual := string(recs[i].Payload.After.Bytes())

		is.Equal(expected, actual)
	}

	is.Equal(
		totalMessages/maxNumberOfMessages,
		receiveMessageCalls,
	) // expected receive calls != actual receive calls made
}
