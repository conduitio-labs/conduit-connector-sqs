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
	"context"
	"fmt"

	"github.com/conduitio-labs/conduit-connector-sqs/common"
	sdk "github.com/conduitio/conduit-connector-sdk"
)

type Config struct {
	sdk.DefaultDestinationMiddleware
	common.Config

	// QueueName is the sqs queue name
	QueueName string `json:"aws.queue" default:"{{ index .Metadata \"opencdc.collection\" }}"`

	// MessageDelay represents the length of time, in seconds, for which a
	// specific message is delayed
	MessageDelay int32 `json:"aws.delayTime"`

	// BatchSize represents the amount of records written per batch
	BatchSize int `json:"batchSize" default:"10"`

	parseQueueName queueNameParser
}

func (config *Config) Validate(ctx context.Context) error {
	switch queue := config.QueueName; {
	case isGoTemplate(queue):
		parser, err := parserFromGoTemplate(queue)
		if err != nil {
			return fmt.Errorf("failed to create template parser: %w", err)
		}
		config.parseQueueName = parser
	case queue != "":
		config.parseQueueName = staticParser(queue)
	default:
		config.parseQueueName = parseAlwaysFromCollection
	}

	return nil
}
