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
	"encoding/base64"
	"testing"
	"time"

	"github.com/conduitio-labs/conduit-connector-sqs/common"
	testutils "github.com/conduitio-labs/conduit-connector-sqs/test"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/matryer/is"
	"go.uber.org/goleak"
)

func TestAcceptance(t *testing.T) {
	is := is.New(t)

	ctx := testutils.TestContext(t)
	sourceConfig := testutils.IntegrationConfig("")
	destinationConfig := testutils.IntegrationConfig("")

	driver := sdk.ConfigurableAcceptanceTestDriver{
		Config: sdk.ConfigurableAcceptanceTestDriverConfig{
			Context:           ctx,
			GoleakOptions:     []goleak.Option{goleak.IgnoreCurrent()},
			Connector:         Connector,
			SourceConfig:      sourceConfig,
			DestinationConfig: destinationConfig,
			BeforeTest: func(t *testing.T) {
				// sqs client creation must be created and cleaned up inside
				// BeforeTest so that goleak doesn't alert of a false positive http
				// connection leaking.
				testClient, closeTestClient := testutils.NewSQSClient(ctx, is)
				defer closeTestClient()

				queue := testutils.CreateTestQueue(ctx, t, is, testClient)
				sourceConfig[common.ConfigAwsQueue] = queue.Name
				destinationConfig[common.ConfigAwsQueue] = queue.Name
			},
			Skip: []string{
				"TestSource_Open_ResumeAtPositionCDC",
				"TestSource_Open_ResumeAtPositionSnapshot",
			},
			WriteTimeout: 500 * time.Millisecond,
			ReadTimeout:  500 * time.Millisecond,
		},
	}

	sdk.AcceptanceTest(t, testDriver{driver})
}

type testDriver struct {
	sdk.ConfigurableAcceptanceTestDriver
}

func (d testDriver) GenerateRecord(t *testing.T, op opencdc.Operation) opencdc.Record {
	rec := d.ConfigurableAcceptanceTestDriver.GenerateRecord(t, op)
	for key := range rec.Metadata {
		val := rec.Metadata[key]
		delete(rec.Metadata, key)

		// sqs is restrictive on the kind of chars allowed as message attributes.
		key = base64.RawURLEncoding.EncodeToString([]byte(key))
		val = base64.RawURLEncoding.EncodeToString([]byte(val))
		rec.Metadata[key] = val
	}

	return rec
}
