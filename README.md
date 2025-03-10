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

<!-- readmegen:source.parameters.yaml -->
```yaml
version: 2.2
pipelines:
  - id: example
    status: running
    connectors:
      - id: example
        plugin: "sqs"
        settings:
          # AWSAccessKeyID is the amazon access key id
          # Type: string
          # Required: yes
          aws.accessKeyId: ""
          # QueueName is the sqs queue name
          # Type: string
          # Required: yes
          aws.queue: ""
          # AWSRegion is the amazon sqs region
          # Type: string
          # Required: yes
          aws.region: ""
          # AWSSecretAccessKey is the amazon secret access key
          # Type: string
          # Required: yes
          aws.secretAccessKey: ""
          # MaxNumberOfMessages is the maximum number of messages to fetch from
          # SQS in a single batch.
          # Type: int
          # Required: no
          aws.maxNumberOfMessages: "1"
          # AWSURL is the URL for AWS (internal use only).
          # Type: string
          # Required: no
          aws.url: ""
          # VisibilityTimeout is the duration (in seconds) that the received
          # messages are hidden from subsequent reads after being retrieved.
          # Type: int
          # Required: no
          aws.visibilityTimeout: "0"
          # WaitTimeSeconds is the duration (in seconds) for which the call
          # waits for a message to arrive in the queue before returning.
          # Type: int
          # Required: no
          aws.waitTimeSeconds: "10"
          # Maximum delay before an incomplete batch is read from the source.
          # Type: duration
          # Required: no
          sdk.batch.delay: "0"
          # Maximum size of batch before it gets read from the source.
          # Type: int
          # Required: no
          sdk.batch.size: "0"
          # Specifies whether to use a schema context name. If set to false, no
          # schema context name will be used, and schemas will be saved with the
          # subject name specified in the connector (not safe because of name
          # conflicts).
          # Type: bool
          # Required: no
          sdk.schema.context.enabled: "true"
          # Schema context name to be used. Used as a prefix for all schema
          # subject names. If empty, defaults to the connector ID.
          # Type: string
          # Required: no
          sdk.schema.context.name: ""
          # Whether to extract and encode the record key with a schema.
          # Type: bool
          # Required: no
          sdk.schema.extract.key.enabled: "false"
          # The subject of the key schema. If the record metadata contains the
          # field "opencdc.collection" it is prepended to the subject name and
          # separated with a dot.
          # Type: string
          # Required: no
          sdk.schema.extract.key.subject: "key"
          # Whether to extract and encode the record payload with a schema.
          # Type: bool
          # Required: no
          sdk.schema.extract.payload.enabled: "false"
          # The subject of the payload schema. If the record metadata contains
          # the field "opencdc.collection" it is prepended to the subject name
          # and separated with a dot.
          # Type: string
          # Required: no
          sdk.schema.extract.payload.subject: "payload"
          # The type of the payload schema.
          # Type: string
          # Required: no
          sdk.schema.extract.type: "avro"
```
<!-- /readmegen:source.parameters.yaml -->

## Destination

The destination connector batches incoming records and pushes them to the Amazon
SQS Queue. Any fields defined in the metadata of the record will be passed as
Message Attributes, and the json encoding of the record will be passed as the
Message Body.

### Configuration

<!-- readmegen:destination.parameters.yaml -->
```yaml
version: 2.2
pipelines:
  - id: example
    status: running
    connectors:
      - id: example
        plugin: "sqs"
        settings:
          # AWSAccessKeyID is the amazon access key id
          # Type: string
          # Required: yes
          aws.accessKeyId: ""
          # AWSRegion is the amazon sqs region
          # Type: string
          # Required: yes
          aws.region: ""
          # AWSSecretAccessKey is the amazon secret access key
          # Type: string
          # Required: yes
          aws.secretAccessKey: ""
          # MessageDelay represents the length of time, in seconds, for which a
          # specific message is delayed
          # Type: int
          # Required: no
          aws.delayTime: "0"
          # QueueName is the sqs queue name
          # Type: string
          # Required: no
          aws.queue: "{{ index .Metadata "opencdc.collection" }}"
          # AWSURL is the URL for AWS (internal use only).
          # Type: string
          # Required: no
          aws.url: ""
          # BatchSize represents the amount of records written per batch
          # Type: int
          # Required: no
          batchSize: "10"
          # Maximum delay before an incomplete batch is written to the
          # destination.
          # Type: duration
          # Required: no
          sdk.batch.delay: "0"
          # Maximum size of batch before it gets written to the destination.
          # Type: int
          # Required: no
          sdk.batch.size: "0"
          # Allow bursts of at most X records (0 or less means that bursts are
          # not limited). Only takes effect if a rate limit per second is set.
          # Note that if `sdk.batch.size` is bigger than `sdk.rate.burst`, the
          # effective batch size will be equal to `sdk.rate.burst`.
          # Type: int
          # Required: no
          sdk.rate.burst: "0"
          # Maximum number of records written per second (0 means no rate
          # limit).
          # Type: float
          # Required: no
          sdk.rate.perSecond: "0"
          # The format of the output record. See the Conduit documentation for a
          # full list of supported formats
          # (https://conduit.io/docs/using/connectors/configuration-parameters/output-format).
          # Type: string
          # Required: no
          sdk.record.format: "opencdc/json"
          # Options to configure the chosen output record format. Options are
          # normally key=value pairs separated with comma (e.g.
          # opt1=val2,opt2=val2), except for the `template` record format, where
          # options are a Go template.
          # Type: string
          # Required: no
          sdk.record.format.options: ""
          # Whether to extract and decode the record key with a schema.
          # Type: bool
          # Required: no
          sdk.schema.extract.key.enabled: "true"
          # Whether to extract and decode the record payload with a schema.
          # Type: bool
          # Required: no
          sdk.schema.extract.payload.enabled: "true"
```
<!-- /readmegen:destination.parameters.yaml -->

## How to use FIFO queues with the connector

Two special metadata keys can be provided to the record to customize how messages are written to FIFO queues.

- `groupID`: It represents the [Amazon SQS message group ID](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/using-messagegroupid-property.html).
- `deduplicationID`: Use this to enforce [SQS exactly-once processing](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/FIFO-queues-exactly-once-processing.html)

There are no special requirements / parameters needed to read from a FIFO queue.
