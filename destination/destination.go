// Copyright © 2022 Meroxa, Inc.
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

	cf "github.com/meroxa/conduit-connector-amazon-sqs/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	sdk "github.com/conduitio/conduit-connector-sdk"
)

type Destination struct {
	sdk.UnimplementedDestination
	config   cf.Config
	svc      *sqs.Client
	queueURL *string
}

func NewDestination() sdk.Destination {
	return sdk.DestinationWithMiddleware(&Destination{}, sdk.DefaultDestinationMiddleware()...)
}

func (d *Destination) Parameters() map[string]sdk.Parameter {
	return map[string]sdk.Parameter{
		cf.ConfigKeyAWSAccessKeyID: {
			Default: "",
			Validations: []sdk.Validation{
				sdk.ValidationRequired{},
			},
			Description: "AWS Access Key ID.",
		},
		cf.ConfigKeyAWSSecretAccessKey: {
			Default: "",
			Validations: []sdk.Validation{
				sdk.ValidationRequired{},
			},
			Description: "AWS Secret Access Key.",
		},
		cf.ConfigKeyAWSToken: {
			Default:     "",
			Description: "AWS Access Token (optional).",
		},
		cf.ConfigKeyAWSQueue: {
			Default: "",
			Validations: []sdk.Validation{
				sdk.ValidationRequired{},
			},
			Description: "AWS SQS Queue Name.",
		},
	}
}

func (d *Destination) Configure(ctx context.Context, cfg map[string]string) error {
	sdk.Logger(ctx).Debug().Msg("Configuring Destination Connector.")
	parsedCfg, err := cf.ParseConfig(cfg)
	if err != nil {
		return err
	}
	d.config = parsedCfg

	return nil
}

func (d *Destination) Open(ctx context.Context) error {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				d.config.AWSAccessKeyID,
				d.config.AWSSecretAccessKey,
				d.config.AWSToken)),
	)
	if err != nil {
		return err
	}
	// Create a SQS client from just a session.
	d.svc = sqs.NewFromConfig(cfg)

	queueInput := &sqs.GetQueueUrlInput{
		QueueName: &d.config.AWSQueue,
	}

	// Get URL of queue
	urlResult, err := d.svc.GetQueueUrl(ctx, queueInput)
	if err != nil {
		return err
	}

	d.queueURL = urlResult.QueueUrl

	return nil
}

func (d *Destination) Write(ctx context.Context, records []sdk.Record) (int, error) {
	for i, record := range records {
		messageBody := string(record.Payload.After.Bytes())
		createdAtTime, err := record.Metadata.GetCreatedAt()
		if err != nil {
			return i, err
		}
		sourcePlugin, err := record.Metadata.GetConduitSourcePluginName()
		if err != nil {
			return i, err
		}
		readAt, err := record.Metadata.GetReadAt()
		if err != nil {
			return i, err
		}

		openCDC, err := record.Metadata.GetOpenCDCVersion()
		if err != nil {
			return i, err
		}
		// construct record to send to destination
		sendMessageInput := &sqs.SendMessageInput{
			DelaySeconds: 10,
			MessageAttributes: map[string]types.MessageAttributeValue{
				"CreatedAt": {
					DataType:    aws.String("String"),
					StringValue: aws.String(createdAtTime.String()),
				},
				"SourcePlugin": {
					DataType:    aws.String("String"),
					StringValue: aws.String(sourcePlugin),
				},
				"ReatAt": {
					DataType:    aws.String("String"),
					StringValue: aws.String(readAt.String()),
				},
				"OpenCDC": {
					DataType:    aws.String("String"),
					StringValue: aws.String(openCDC),
				},
			},
			MessageBody: &messageBody,
			QueueUrl:    d.queueURL,
		}

		_, err = d.svc.SendMessage(ctx, sendMessageInput)
		if err != nil {
			return i, err
		}
	}

	return len(records), nil
}

func (d *Destination) Teardown(_ context.Context) error {
	return nil
}
