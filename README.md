# CDP Gateway - Batch Collector

The batch collector is a NATS consumer that consumes all messages on a given subject, zips up batches of messages, and forwards the zipped messages to another subject. It is used to reduce the chattiness of streams that are going to be forwarded along to remote environments, such as over the public internet to a cloud storage provider.

The collector requires a configured batch size `N`, such that a zipped message will be published for each `N`
messages received. A timeout is also required, which may be any valid golang duration (https://pkg.go.dev/time@go1.16.6#ParseDuration). After this duration has elapsed, and less then `N` messages are received, a partial batch will be sent if there are `M` messages have been received such that 0 < `M` < `N`

## Docker Build

`gradle dockerBuild`

The `dockerBuild` gradle task will cross compile the executable for linux/amd64,
and build the `Dockerfile`, tagging `gateway-batch-collector:latest`.

If you want to diy it for some reason, the following will work, but be sure to have
built a linux binary beforehand and placed it at `build/collector`

`docker build . -t gateway-batch-collector:latest`

## Executable Build

`gradle build`

Builds a native executable for the current platform, unless you set `GOOS` and/or `GOARCH`
environment variables. `gradle goBuildLinux64` is provided as a convenience task to create
a linux/amd64 build on mac or windows

## Local Dev/Test Deployment

`cd testing`
`docker-compose up`

will bring up a single nats container with jetstream enabled, a nats tools container that will
configure the NATS jetstrea by creating the appropriate streams and consumers, and a running batch collector
connected to the same nats instance. Ports are forwarded so that the nats cli utility can
be used to observe stream and consumer information or publish test messages.

### Configuration

All collector configuration is done via environment variables. Do not change `NATS_SERVER_URL` when using docker-compose,
but other vars can be configured in `docker-compose.yml`. The current default configuration is:

```
      NATS_SERVER_URL: "nats://nats-main:4222"
      NATS_INCOMING_SUBJECT: "HL7.incoming"
      NATS_OUTGOING_SUBJECT: "HL7.batched"
      MSG_BATCH_SIZE: "100"
      MSG_BATCH_TIMEOUT: "60s"
```

### Test Procedures

The nats cli provides an easy way to test the batch and timeout features by publishing with `nats pub`.

If you have not used the nats cli previously then you will need to install using the following commands (mac):

```
brew tap nats-io/nats-tools
brew install nats-io/nats-tools/nats
```

Then you can use the following command to send messages:

```
nats pub HL7.incoming "message {{.Count}} @ {{.TimeStamp}}" --count=10101
```

The above publishes 10,101 messages to `HL7.incoming`. With a batch size of 100 and a 60s timeout, you
should see 101 full batches right away, and then a single message 60 seconds later
