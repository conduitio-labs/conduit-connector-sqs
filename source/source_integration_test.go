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
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/conduitio-labs/conduit-connector-sqs/common"
	testutils "github.com/conduitio-labs/conduit-connector-sqs/test"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/matryer/is"
)

func TestSource_SuccessfulMessageReceive(t *testing.T) {
	is := is.New(t)
	ctx := testutils.TestContext(t)
	source := NewSource()
	defer func() {
		err := source.Teardown(ctx)
		is.NoErr(err)
	}()

	testClient := testutils.NewSQSClient(ctx, is)
	testQueue := testutils.CreateTestQueue(ctx, t, is, testClient)
	cfg := testutils.IntegrationConfig(testQueue.Name)

	messageBody := "Test message body"
	_, err := testClient.SendMessage(
		ctx,
		&sqs.SendMessageInput{
			MessageBody: &messageBody,
			QueueUrl:    testQueue.URL,
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

	is.NoErr(source.Teardown(ctx))
}

func TestSource_FailBadCreds(t *testing.T) {
	is := is.New(t)
	ctx := testutils.TestContext(t)
	source := NewSource()
	defer func() {
		err := source.Teardown(ctx)
		is.NoErr(err)
	}()

	testClient := testutils.NewSQSClient(ctx, is)
	testQueue := testutils.CreateTestQueue(ctx, t, is, testClient)
	cfg := testutils.IntegrationConfig(testQueue.Name)

	cfg[common.ConfigKeyAWSAccessKeyID] = ""

	err := source.Configure(ctx, cfg)
	is.NoErr(err)

	err = source.Open(ctx, nil)
	is.True(strings.Contains(err.Error(), "failed to refresh cached credentials, static credentials are empty"))
}

func TestSource_EmptyQueue(t *testing.T) {
	is := is.New(t)
	ctx := testutils.TestContext(t)

	testClient := testutils.NewSQSClient(ctx, is)
	testQueue := testutils.CreateTestQueue(ctx, t, is, testClient)
	cfg := testutils.IntegrationConfig(testQueue.Name)

	source := NewSource()
	defer func() { is.NoErr(source.Teardown(ctx)) }()

	err := source.Configure(ctx, cfg)
	is.NoErr(err)

	err = source.Open(ctx, nil)
	is.NoErr(err)

	record, err := source.Read(ctx)

	if err != sdk.ErrBackoffRetry || record.Metadata != nil {
		t.Fatalf("expected no records and a signal that there are no more records, got %v %v", record, err)
	}
}
