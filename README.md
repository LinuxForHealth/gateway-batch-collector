# CDP Gateway - Batch Collector

The batch collector is a NATS consumer that consumes all messages on a given subject, zips up batches of messages, and forwards the zipped messages to another subject. It is used to reduce the chattiness of streams that are going to be forwarded along to remote environments, such as over the public internet to a cloud storage provider.

The collector requires a configured batch size `N`, such that a zipped message will be published for each `N`
messages received. A timeout is also required, which may be any valid golang duration (https://pkg.go.dev/time@go1.16.6#ParseDuration). When no messages have been received for this duration, a partial batch will be sent if there
are `M` messages have been received such that 0 < `M` < `N`

## Docker Build

```gradle dockerBuild```

The `dockerBuild` gradle task will cross compile the executable for linux/amd64,
and build the `Dockerfile`, tagging `whpa-cdp-gateway-batch-collector:latest`.

If you want to diy it for some reason, the following will work, but be sure to have
built a linux binary beforehand and placed it at `build/collector`

```docker build . -t whpa-cdp-gateway-batch-collector:latest```

## Executable Build

```gradle goBuild```

Builds a native executable for the current platform, unless you set `GOOS` and/or `GOARCH`
environment variables. `gradle goBuildLinux64` is provided as a convenience task to create
a linux/amd64 build on mac or windows

## Local Dev/Test Deployment

```docker-compose up```

will bring up a single nats container with jetstream enabled and a running batch collector
connected to the same nats instance. Ports are forwarded so that the nats cli utility can
be used to observe stream and consumer information or publish test messages.

### Configuration

All collector configuration is done via environment variables. Do not change `NATS_URL` when using docker-compose,
but other vars can be configured in `docker-compose.yml`. The current default configuration is:

```
      NATS_URL: "nats://nats-main:4222"
      NATS_STREAM_NAME: "HL7"
      NATS_INCOMING_SUBJECT_NAME: "HL7.incoming"
      NATS_OUTGOING_SUBJECT_NAME: "HL7.batched"
      NATS_QUEUE_NAME: ""
      MSG_BATCH_SIZE: "100"
      MSG_BATCH_TIMEOUT: "60s"
      NATS_DURABLE_NAME: "batchCollector"
```

`NATS_STREAM_NAME`, when set, causes the collector to create the stream if it doesn't exist. Unset this
variable if your stream configuration will be done ahead of time by another process. In that case, the
collector process will terminate if the necessary streams do not exist for the configured subjects.

Setting `NATS_QUEUE_NAME` will add the collector to a queue group (https://docs.nats.io/nats-concepts/queue)

Setting `NATS_DURABLE_NAME` allows the nats server to track the collector so that everything can pick back up after
an interruption due to a crash or restart. However, when using this setting, configuration changes may require the old
consumer configuration to be removed with `nats consumer rm`. When using docker-compose, you may want to just delete the
containers and recreate them when making changes related to consumer configuration.

### Test Procedures

The nats cli provides an easy way to test the batch and timeout features by publishing with `nats pub`.

```
nats pub HL7.incoming "message {{.Count}} @ {{.TimeStamp}}" --count=10101
```

The above publishes 10,101 messages to `HL7.incoming`. With a batch size of 100 and a 60s timeout, you
should see 101 full batches right away, and then a single message 60 seconds later 