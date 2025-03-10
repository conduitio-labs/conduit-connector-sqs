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

package sqs

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/conduitio-labs/conduit-connector-sqs/destination"
	"github.com/conduitio-labs/conduit-connector-sqs/source"
	"github.com/conduitio-labs/conduit-connector-sqs/spec"
	testutils "github.com/conduitio-labs/conduit-connector-sqs/test"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/matryer/is"
)

func TestFifoQueues(t *testing.T) {
	is := is.New(t)
	ctx := testutils.TestContext(t)

	testClient, cleanTestClient := testutils.NewSQSClient(ctx, is)
	defer cleanTestClient()

	testQueue := testutils.CreateTestFifoQueue(ctx, t, is, testClient)

	src, cleanSrc := testutils.StartSource(ctx, is, source.NewSource(), testQueue.Name)
	defer cleanSrc()

	dest, cleanDest := testutils.StartDestination(ctx, is, destination.NewDestination(), testQueue.Name)
	defer cleanDest()

	var recs []opencdc.Record
	for i := 1; i <= 10; i++ {
		rec := sdk.Util.Source.NewRecordCreate(
			opencdc.Position(nil), // doesn't matter
			opencdc.Metadata{
				destination.GroupIDKey: "test-group-id",
				destination.DedupIDKey: fmt.Sprint("dedup-key-", i),
			},
			opencdc.RawData(fmt.Sprint("key-", i)),
			opencdc.RawData(fmt.Sprint("val-", i)),
		)

		recs = append(recs, rec)
	}

	// write a duplicated record to test for sqs dedup functionality
	recs = append(recs, sdk.Util.Source.NewRecordCreate(
		opencdc.Position(nil), // doesn't matter
		opencdc.Metadata{
			destination.GroupIDKey: "test-group-id",
			destination.DedupIDKey: "dedup-key-10",
		},
		opencdc.RawData("key-10"),
		opencdc.RawData("val-10"),
	))

	_, err := dest.Write(ctx, recs)
	is.NoErr(err)

	for i := 1; i <= 10; i++ {
		rec, err := src.Read(ctx)
		is.NoErr(err)

		var parsed opencdc.Record
		is.NoErr(json.Unmarshal(rec.Payload.After.Bytes(), &parsed))

		actual := string(parsed.Payload.After.Bytes())
		expected := fmt.Sprint("val-", i)
		is.Equal(actual, expected)
		is.Equal(parsed.Metadata[destination.GroupIDKey], "test-group-id")
		is.Equal(parsed.Metadata[destination.DedupIDKey], fmt.Sprint("dedup-key-", i))

		is.NoErr(src.Ack(ctx, rec.Position))
	}

	// test that the additional duplicated record wasn't written
	_, err = src.Read(ctx)
	is.Equal(err, sdk.ErrBackoffRetry)
}

type multicollectionTestCase struct {
	QueueName       string
	ExpectedRecords []opencdc.Record
}

func (testCase multicollectionTestCase) eval(ctx context.Context, is *is.I) {
	source := source.NewSource()
	cfg := testutils.SourceConfig(testCase.QueueName)

	err := sdk.Util.ParseConfig(ctx, cfg, source.Config(), spec.Spec().SourceParams)
	is.NoErr(err)

	is.NoErr(source.Open(ctx, nil))

	for _, expectedRec := range testCase.ExpectedRecords {
		rec, err := source.Read(ctx)
		is.NoErr(err)

		var actualRec opencdc.Record
		is.NoErr(json.Unmarshal(rec.Payload.After.Bytes(), &actualRec))

		is.Equal(cmp.Diff(expectedRec, actualRec, cmpopts.IgnoreUnexported(
			expectedRec, actualRec,
		)), "")

		is.NoErr(source.Ack(ctx, rec.Position))
	}

	is.NoErr(source.Teardown(ctx))
}

func TestMulticollection_MultipleQueues(t *testing.T) {
	is := is.New(t)
	ctx := testutils.TestContext(t)

	testClient, closeTestClient := testutils.NewSQSClient(ctx, is)
	defer closeTestClient()

	testQueue1 := testutils.CreateTestQueue(ctx, t, is, testClient)
	testQueue2 := testutils.CreateTestQueue(ctx, t, is, testClient)
	testQueue3 := testutils.CreateTestQueue(ctx, t, is, testClient)

	queueName := "" // empty to force fetch from `opencdc.collection` metadata field

	destination, cleanDestination := testutils.StartDestination(
		ctx, is, destination.NewDestination(),
		queueName,
	)
	defer cleanDestination()

	genRecord := func(queueName string) opencdc.Record {
		rec := opencdc.Record{
			Position:  nil,
			Operation: opencdc.OperationCreate,
			Key:       opencdc.RawData(uuid.NewString()),
			Payload: opencdc.Change{
				Before: opencdc.StructuredData(nil),
				After:  opencdc.StructuredData(nil),
			},
		}

		rec.Metadata = opencdc.Metadata{}
		rec.Metadata.SetCollection(queueName)
		return rec
	}

	recs := []opencdc.Record{
		genRecord(testQueue1.Name),
		genRecord(testQueue1.Name),
		genRecord(testQueue2.Name),
		genRecord(testQueue2.Name),
		genRecord(testQueue3.Name),
		genRecord(testQueue3.Name),
	}

	written, err := destination.Write(ctx, recs)
	is.NoErr(err)
	is.Equal(written, len(recs))

	for _, testCase := range []multicollectionTestCase{
		{QueueName: testQueue1.Name, ExpectedRecords: recs[:2]},
		{QueueName: testQueue2.Name, ExpectedRecords: recs[2:4]},
		{QueueName: testQueue3.Name, ExpectedRecords: recs[4:6]},
	} {
		testCase.eval(ctx, is)
	}
}

func TestMulticollection_QueueNameAsTemplate(t *testing.T) {
	is := is.New(t)
	ctx := testutils.TestContext(t)

	testClient, closeTestClient := testutils.NewSQSClient(ctx, is)
	defer closeTestClient()

	testQueue1 := testutils.CreateTestQueue(ctx, t, is, testClient)
	testQueue2 := testutils.CreateTestQueue(ctx, t, is, testClient)

	template := `{{ index .Metadata "sqsQueueName" }}`

	destination, cleanDestination := testutils.StartDestination(
		ctx, is, destination.NewDestination(),
		template,
	)
	defer cleanDestination()

	genRecord := func(queueName string) opencdc.Record {
		rec := opencdc.Record{
			Position:  nil,
			Operation: opencdc.OperationCreate,
			Key:       opencdc.RawData(uuid.NewString()),
			Payload: opencdc.Change{
				Before: opencdc.StructuredData(nil),
				After:  opencdc.StructuredData(nil),
			},
		}

		rec.Metadata = opencdc.Metadata{
			"sqsQueueName": queueName,
		}
		return rec
	}

	recs := []opencdc.Record{
		genRecord(testQueue1.Name),
		genRecord(testQueue2.Name),
	}

	written, err := destination.Write(ctx, recs)
	is.NoErr(err)
	is.Equal(written, 2)

	for _, testCase := range []multicollectionTestCase{
		{QueueName: testQueue1.Name, ExpectedRecords: recs[:1]},
		{QueueName: testQueue2.Name, ExpectedRecords: recs[1:]},
	} {
		testCase.eval(ctx, is)
	}
}
