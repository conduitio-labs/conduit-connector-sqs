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
	"github.com/conduitio-labs/conduit-connector-sqs/common"
	sdk "github.com/conduitio/conduit-connector-sdk"
)

type Config struct {
	sdk.DefaultSourceMiddleware

	common.Config

	// QueueName is the sqs queue name
	QueueName string `json:"aws.queue" validate:"required"`

	// VisibilityTimeout is the duration (in seconds) that the received messages
	// are hidden from subsequent reads after being retrieved.
	VisibilityTimeout int32 `json:"aws.visibilityTimeout"`

	// WaitTimeSeconds is the duration (in seconds) for which the call waits for
	// a message to arrive in the queue before returning.
	WaitTimeSeconds int32 `json:"aws.waitTimeSeconds" default:"10"`

	// MaxNumberOfMessages is the maximum number of messages to fetch from SQS in a single batch.
	MaxNumberOfMessages int32 `json:"aws.maxNumberOfMessages" default:"1" validate:"gt=0,lt=11"`
}
