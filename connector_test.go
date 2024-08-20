package sqs

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/conduitio-labs/conduit-connector-sqs/common"
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
	srcCfg[common.SourceConfigAwsVisibilityTimeout] = "1"
	srcCfg[common.SourceConfigAwsWaitTimeSeconds] = "2"
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
	for i := 0; i < 10; i++ {
		rec := sdk.Util.Source.NewRecordCreate(
			opencdc.Position(nil), // doesn't matter
			opencdc.Metadata{
				destination.GroupIDKey: "test-group-id",
			},
			opencdc.RawData(fmt.Sprint("key-", i)),
			opencdc.RawData(fmt.Sprint("val-", i)),
		)

		recs = append(recs, rec)
	}

	_, err := dest.Write(ctx, recs)
	is.NoErr(err)

	for i := 0; i < len(recs); i++ {
		rec, err := src.Read(ctx)
		is.NoErr(err)

		var parsed opencdc.Record
		is.NoErr(json.Unmarshal(rec.Payload.After.Bytes(), &parsed))

		actual := string(parsed.Payload.After.Bytes())
		expected := fmt.Sprint("val-", i)
		is.Equal(actual, expected)

		src.Ack(ctx, rec.Position)
	}
}
