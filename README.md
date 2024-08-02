# Conduit Connector for Amazon SQS

[Conduit](https://conduit.io) for [Amazon SQS](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/welcome.html).

## How to build?

Run `make build` to build the connector.

## Source

The source connector pulls data from the Amazon SQS Queue. As messages come in, the source connector grabs a single message from the Amazon SQS Queue, formats it for the destination connector as a new record, and sends it. The Message Body of the SQS Message is formatted into a record's payload, and the Message Attributes are passed as record metadata. After the record has been acknowledged, the source connector deletes the message from the Amazon SQS Queue.

## Destinaton

The destination connector batches incoming records into 10 and pushes them to the Amazon SQS Queue. Any fields defined in the metadata of the record will be passed as Message Attributes, and the json encoding of the record will be passed as the Message Body.

### Configuration

The configuration passed to `Configure` can contain the following fields:

#### Source

| name                  | description                                    | required | example             |
| --------------------- | ---------------------------------------------- | -------- | ------------------- |
| `aws.accessKeyId`     | AWS Access Key ID                              | yes      | "THE_ACCESS_KEY_ID" |
| `aws.secretAccessKey` | AWS Secret Access Key                          | yes      | "SECRET_ACCESS_KEY" |
| `aws.region`          | AWS SQS Region                                 | yes      | "us-east-1"         |
| `aws.queue`           | AWS SQS Queue Name                             | yes      | "QUEUE_NAME"        |
| `aws.delayTime`       | AWS SQS Message Delay                          | yes      | "5"                 |
| `aws.url`             | URL for AWS (internal use only)  | no       |                     |

#### Destination

| name                    | description                                    | required | example             |
| ----------------------- | ---------------------------------------------- | -------- | ------------------- |
| `aws.accessKeyId`       | AWS Access Key ID                              | yes      | "THE_ACCESS_KEY_ID" |
| `aws.secretAccessKey`   | AWS Secret Access Key                          | yes      | "SECRET_ACCESS_KEY" |
| `aws.region`            | AWS SQS Region                                 | yes      | "us-east-1"         |
| `aws.queue`             | AWS SQS Queue Name                             | yes      | "QUEUE_NAME"        |
| `aws.visibilityTimeout` | AWS SQS Message Visibility Timeout             | yes      | "5"                 |
| `aws.url`               | AWSURL is the URL for AWS (internal use only). | no       |                     |
