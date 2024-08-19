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

package common

type Config struct {
	// AWSAccessKeyID is the amazon access key id
	AWSAccessKeyID string `json:"aws.accessKeyId" validate:"required"`
	// AWSSecretAccessKey is the amazon secret access key
	AWSSecretAccessKey string `json:"aws.secretAccessKey" validate:"required"`
	// AWSRegion is the amazon sqs region
	AWSRegion string `json:"aws.region" validate:"required"`
	// AWSQueue is the sqs queue name
	AWSQueue string `json:"aws.queue" validate:"required"`
	// AWSURL is the URL for AWS (internal use only).
	AWSURL string `json:"aws.url"`
}

//go:generate paramgen -output=source_paramgen.go SourceConfig

type SourceConfig struct {
	Config
	// VisibilityTimeout is the duration (in seconds) that the received messages
	// are hidden from subsequent reads after being retrieved.
	VisibilityTimeout int32 `json:"aws.visibilityTimeout" default:"0"`

	// WaitTimeSeconds is the duration (in seconds) for which the call waits for
	// a message to arrive in the queue before returning.
	WaitTimeSeconds int32 `json:"aws.waitTimeSeconds" default:"10"`
}

//go:generate paramgen -output=destination_paramgen.go DestinationConfig

type DestinationConfig struct {
	Config
	// MessageDelay represents the length of time, in seconds, for which a
	// specific message is delayed
	MessageDelay int32 `json:"aws.delayTime" default:"0"`
}
