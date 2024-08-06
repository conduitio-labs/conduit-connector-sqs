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
	"testing"
	"time"

	"github.com/conduitio-labs/conduit-connector-sqs/common"
	"github.com/conduitio-labs/conduit-connector-sqs/source"
	testutils "github.com/conduitio-labs/conduit-connector-sqs/test"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/matryer/is"
	"go.uber.org/goleak"
)

func TestAcceptance(t *testing.T) {
	is := is.New(t)

	ctx := testutils.TestContext(t)
	sourceConfig := testutils.IntegrationConfig("")
	destinationConfig := testutils.IntegrationConfig("")

	sdk.AcceptanceTest(t, sdk.ConfigurableAcceptanceTestDriver{
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
				sourceConfig[common.ConfigKeyAWSQueue] = queue.Name
				sourceConfig[source.ConfigKeySQSVisibilityTimeout] = "1"
				destinationConfig[common.ConfigKeyAWSQueue] = queue.Name

				sdk.Logger(ctx).Info().Msgf("queue name: %v", queue.Name)
				sdk.Logger(ctx).Info().Msgf("queue url: %v", *queue.URL)
			},
			Skip: []string{
				"TestSource_Configure_RequiredParams",
				"TestDestination_Configure_RequiredParams",
				// "TestDestination_Configure_Success",
				// "TestDestination_Parameters_Success",
				// "TestDestination_Write_Succes",
				// "TestSource_Configure_RequiredParams",
				// "TestSource_Configure_Success",
				// "TestSource_Open_ResumeAtPositionCDC",
				// "TestSource_Open_ResumeAtPositionSnapshot",
				// "TestSource_Parameters_Success",
				// "TestSource_Read_Success",
				// "TestSource_Read_Timeout",
				// "TestSpecifier_Exists",
				// "TestSpecifier_Specify_Success",
			},
			WriteTimeout: 5000 * time.Millisecond,
			ReadTimeout:  5000 * time.Millisecond,
		},
	})
}
