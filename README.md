# Conduit Connector for Amazon SQS

[Conduit](https://conduit.io) for [Amazon SQS](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/welcome.html).

## How to build?

Run `make build` to build the connector.

## Source

The source connector pulls data from the Amazon SQS Queue. As messages come in, the source connector grabs a single message from the Amazon SQS Queue, formats it for the destination connector as a new record, and sends it. The Message Body of the SQS Message is formatted into a record's payload, and the Message Attributes are passed as record metadata. After the record has been acknowledged, the source connector deletes the message from the Amazon SQS Queue.

## Destination

The destination connector batches incoming records into 10 and pushes them to the Amazon SQS Queue. Any fields defined in the metadata of the record will be passed as Message Attributes, and the json encoding of the record will be passed as the Message Body.

It can work in three modes, depending of the `aws.queue` parameter:

- with a queue name: the connector will obtain the queue url from the given sqs queue name, and write records there.
- with a go template: the connector will evaluate the given go template against each record to obtain the queue name.
- empty queue name: the connector will try to obtain the queue name from the `opencdc.collection` metadata field.

It supports multicollection mode, where the sqs queue name is determined at runtime based on the `opencdc.collection` metadata field or a given go template. If no queue name is found, the record will be written to the specified `aws.queue` parameter.

### Configuration

The configuration passed to `Configure` can contain the following fields:

#### Source

| name                  | description                     | required | example             |
| --------------------- | ------------------------------- | -------- | ------------------- |
| `aws.accessKeyId`     | AWS Access Key ID               | yes      | "THE_ACCESS_KEY_ID" |
| `aws.secretAccessKey` | AWS Secret Access Key           | yes      | "SECRET_ACCESS_KEY" |
| `aws.region`          | AWS SQS Region                  | yes      | "us-east-1"         |
| `aws.queue`           | AWS SQS Queue Name              | yes      | "QUEUE_NAME"        |
| `aws.delayTime`       | AWS SQS Message Delay           | yes      | "5"                 |
| `aws.url`             | URL for AWS (internal use only) | no       |                     |

#### Destination

| name                              | description                                                            | required | example                                      |
| --------------------------------- | ---------------------------------------------------------------------- | -------- | -------------------------------------------- |
| `aws.accessKeyId`                 | AWS Access Key ID                                                      | yes      | "THE_ACCESS_KEY_ID"                          |
| `aws.secretAccessKey`             | AWS Secret Access Key                                                  | yes      | "SECRET_ACCESS_KEY"                          |
| `aws.region`                      | AWS SQS Region                                                         | yes      | "us-east-1"                                  |
| `aws.queue` (as a sqs queue name) | AWS queue name. Records will be written there                          | no       | `my-sqs-queue`                               |
| `aws.queue` (as a go template)    | Go template that is evaluated at runtime to obtain the queue name      | no       | `{{ index .Metadata "opencdc.collection" }}` |
| `aws.queue` (empty)               | Makes the connector fetch the queue name from the `opencdc.collection` | no       | ""                                           |
| `aws.visibilityTimeout`           | AWS SQS Message Visibility Timeout                                     | yes      | "5"                                          |
| `aws.url`                         | URL for AWS (internal use only)                                        | no       |                                              |

## How to use FIFO queues with the connector

Two special metadata keys can be provided to the record to customize how messages are written to FIFO queues.

- `groupID`: It represents the [Amazon SQS message group ID](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/using-messagegroupid-property.html).
- `deduplicationID`: Use this to enforce [SQS exactly-once processing](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/FIFO-queues-exactly-once-processing.html)

There are no special requirements / parameters needed to read from a FIFO queue.
