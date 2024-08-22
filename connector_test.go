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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/conduitio-labs/conduit-connector-sqs/destination"
	"github.com/conduitio-labs/conduit-connector-sqs/source"
	testutils "github.com/conduitio-labs/conduit-connector-sqs/test"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/matryer/is"
)

func TestFifoQueues(t *testing.T) {
	is := is.New(t)
	ctx := testutils.TestContext(t)

	testClient, cleanTestClient := testutils.NewSQSClient(ctx, is)
	defer cleanTestClient()

	testQueue := testutils.CreateTestFifoQueue(ctx, t, is, testClient)
	srcCfg := testutils.SourceConfig(testQueue.Name)
	srcCfg[source.ConfigAwsVisibilityTimeout] = "1"
	srcCfg[source.ConfigAwsWaitTimeSeconds] = "2"
	destCfg := testutils.DestinationConfig(testQueue.Name)

	src := source.NewSource()
	is.NoErr(src.Configure(ctx, srcCfg))
	is.NoErr(src.Open(ctx, nil))
	defer func() { is.NoErr(src.Teardown(ctx)) }()

	dest := destination.NewDestination()
	is.NoErr(dest.Configure(ctx, destCfg))
	is.NoErr(dest.Open(ctx))
	defer func() { is.NoErr(dest.Teardown(ctx)) }()

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

	_, err = src.Read(ctx)
	is.Equal(err, sdk.ErrBackoffRetry)
}