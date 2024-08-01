// Copyright © 2023 Meroxa, Inc.
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

import (
	"context"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	transport "github.com/aws/smithy-go/endpoints"
)

// EndpointResolver sets a custom endpoint for the kinesis client. It satisfies the
// sqs.EndpointResolverV2 interface.
type EndpointResolver struct{ url url.URL }

func NewEndpointResolver(urlStr string) (sqs.EndpointResolverV2, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint url: %w", err)
	}
	return &EndpointResolver{*u}, nil
}

func (e *EndpointResolver) ResolveEndpoint(
	_ context.Context,
	_ sqs.EndpointParameters,
) (transport.Endpoint, error) {
	return transport.Endpoint{URI: e.url}, nil
}

func NewSQSClient(ctx context.Context, cfg Config) (*sqs.Client, error) {
	sqsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.AWSRegion),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AWSAccessKeyID,
				cfg.AWSSecretAccessKey,
				"")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config with given credentials: %w", err)
	}

	var sqsOptions []func(*sqs.Options)
	if url := cfg.AWSURL; url != "" {
		endpointResolver, err := NewEndpointResolver(url)
		if err != nil {
			return nil, fmt.Errorf("failed to create endpoint resolver: %w", err)
		}

		sqsOptions = append(sqsOptions, sqs.WithEndpointResolverV2(endpointResolver))
	}

	return sqs.NewFromConfig(sqsCfg, sqsOptions...), nil
}
