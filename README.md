# Conduit Connector for Amazon SQS 
[Conduit](https://conduit.io) for [Amazon SQS](https://documentation.spire.com/maritime-2-0/).

## How to build?
Run `make build` to build the connector.


## Source
The source connector pulls data from the Amazon SQS Queue. As messages come in, the source connector grabs a single message from the Amazon SQS Queue, formats it for the destination connector as a new record, and sends it. After the destination connector acknowledges that it has received the record, the source connector deletes the message from the Amazon SQS Queue.


## Destinaton


The destination connector formats incoming records as Amazon SQS Messages and pushes them to the queue. Any fields defined in the metadata of the record will be passed as message attributes, and the payload of the record will be passed as the message body.


### Configuration


The configuration passed to `Configure` can contain the following fields:

#### Source 

| name                  | description                                                                           | required | example             |
| --------------------- | ------------------------------------------------------------------------------------- | -------- | ------------------- |
| `aws.accessKeyId`     | AWS access key id                                                                     | yes      | "THE_ACCESS_KEY_ID" |
| `aws.secretAccessKey` | AWS secret access key                                                                 | yes      | "SECRET_ACCESS_KEY" |
| `aws.region`          | AWS SQS Region                                                                        | yes      | "us-east-1"         |
| `aws.queue`           | AWS SQS Queue Name                                                                    | yes      | "QUEUE_NAME"        |
| `aws.delayTime`       | AWS SQS message delay                                                                 | yes      | 
"5"                 | 

#### Destination

| name                  | description                                                                           | required | example             |
| --------------------- | ------------------------------------------------------------------------------------- | -------- | ------------------- |
| `aws.accessKeyId`     | AWS access key id                                                                     | yes      | "THE_ACCESS_KEY_ID" |
| `aws.secretAccessKey` | AWS secret access key                                                                 | yes      | "SECRET_ACCESS_KEY" |
| `aws.region`          | AWS SQS Region                                                                        | yes      | "us-east-1"         |
| `aws.queue`           | AWS SQS Queue Name                                                                    | yes      | "QUEUE_NAME"        |
| `aws.visibilityTimeout`           | AWS SQS message visibility timeout                                        | yes      |         "5"                 |
