package destination

import (
	"testing"

	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/matryer/is"
)

func recWithMetadata(queueName ...string) opencdc.Record {
	m := make(opencdc.Metadata)
	if len(queueName) != 0 {
		m.SetCollection(queueName[0])
	}
	return sdk.Util.Source.NewRecordCreate(nil, m, nil, nil)
}

func TestSplitIntoBatches_Empty(t *testing.T) {
	is := is.New(t)
	recs := []opencdc.Record{}

	batches := splitIntoBatches(recs, "test-queue")
	is.Equal(len(batches), 0)
}

func TestSplitIntoBatches_MultipleBatches(t *testing.T) {
	is := is.New(t)
	recs := []opencdc.Record{
		recWithMetadata(),
		recWithMetadata(),
		recWithMetadata("test-queue-col-1"),
		recWithMetadata("test-queue-col-2"),
		recWithMetadata(),
		recWithMetadata("test-queue-col-3"),
	}

	batches := splitIntoBatches(recs, "test-queue-default")
	is.Equal(len(batches), 5)
	is.Equal(batches, []messageBatch{
		{
			queueName: "test-queue-default",
			records:   recs[:2],
		},
		{
			queueName: "test-queue-col-1",
			records:   recs[2:3],
		},
		{
			queueName: "test-queue-col-2",
			records:   recs[3:4],
		},
		{
			queueName: "test-queue-default",
			records:   recs[4:5],
		},
		{
			queueName: "test-queue-col-3",
			records:   recs[5:6],
		},
	})
}
