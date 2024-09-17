# Conduit Connector for Amazon SQS

[Conduit](https://conduit.io) for [Amazon SQS](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/welcome.html).

## How to build?

Run `make build` to build the connector.

## Source

The source connector pulls data from the Amazon SQS Queue. As messages come in,
the source connector grabs a single message from the Amazon SQS Queue, formats
it for the destination connector as a new record, and sends it. The Message Body
of the SQS Message is formatted into a record's payload, and the Message
Attributes are passed as record metadata. After the record has been
acknowledged, the source connector deletes the message from the Amazon SQS
Queue.

### Configuration

The configuration passed to `Configure` can contain the following fields:

| name                      | description                                                                                                  | required | default value |
| ------------------------- | ------------------------------------------------------------------------------------------------------------ | -------- | ------------- |
| `aws.accessKeyId`         | AWS Access Key ID                                                                                            | yes      |               |
| `aws.secretAccessKey`     | AWS Secret Access Key                                                                                        | yes      |               |
| `aws.region`              | AWS SQS Region                                                                                               | yes      |               |
| `aws.queue`               | AWS SQS Queue Name                                                                                           | yes      |               |
| `aws.visibilityTimeout`   | The duration (in seconds) that the received messages are hidden from subsequent reads after being retrieved. | no       | 0             |
| `aws.waitTimeSeconds`     | The duration (in seconds) for which the call waits for a message to arrive in the queue before returning.    | no       | 10            |
| `aws.maxNumberOfMessages` | The maximum number of messages to fetch from SQS in a single batch.                                          | no       | 1             |
| `aws.url`                 | URL for AWS (internal use only)                                                                              | no       |               |

## Destination

The destination connector batches incoming records and pushes them to the Amazon
SQS Queue. Any fields defined in the metadata of the record will be passed as
Message Attributes, and the json encoding of the record will be passed as the
Message Body.

### Configuration

| name                  | description                                                                                                                                                                                                                            | required | default value                                |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | -------------------------------------------- |
| `aws.accessKeyId`     | AWS Access Key ID                                                                                                                                                                                                                      | yes      |                                              |
| `aws.secretAccessKey` | AWS Secret Access Key                                                                                                                                                                                                                  | yes      |                                              |
| `aws.region`          | AWS SQS Region                                                                                                                                                                                                                         | yes      |                                              |
| `aws.queue`           | AWS SQS Queue Name. It can contain a [Go template](https://pkg.go.dev/text/template) that will be executed for each record to determine the queue name. By default, the queue is the value of the `opencdc.collection` metadata field. | no       | `{{ index .Metadata "opencdc.collection" }}` |
| `aws.delayTime`       | Represents the length of time, in seconds, for which aspecific message is delayed                                                                                                                                                      | no       | 0                                            |
| `aws.url`             | URL for AWS (internal use only)                                                                                                                                                                                                        | no       |                                              |

## How to use FIFO queues with the connector

Two special metadata keys can be provided to the record to customize how messages are written to FIFO queues.

- `groupID`: It represents the [Amazon SQS message group ID](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/using-messagegroupid-property.html).
- `deduplicationID`: Use this to enforce [SQS exactly-once processing](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/FIFO-queues-exactly-once-processing.html)

There are no special requirements / parameters needed to read from a FIFO queue.
