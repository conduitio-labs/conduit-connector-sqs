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

	dest := &Destination{
		parseQueueName: parserFromCollectionWithDefault("test-queue"),
	}

	batches, err := dest.splitIntoBatches(recs)
	is.NoErr(err)
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

	dest := &Destination{
		parseQueueName: parserFromCollectionWithDefault("test-queue-default"),
	}

	batches, err := dest.splitIntoBatches(recs)
	is.Equal(len(batches), 5)
	is.NoErr(err)

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