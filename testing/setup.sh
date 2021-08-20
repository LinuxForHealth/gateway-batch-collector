#!/bin/sh

nats context add --user=ruser --password=T0pS3cr3t -s nats-main nats
nats context select nats
nats str add HL7 --subjects "HL7.*" --ack --max-msgs=-1 --max-bytes=-1 --max-age=1y --storage file --retention limits --max-msg-size=-1 --discard=old --dupe-window=2m --replicas=1
nats con add HL7 MESSAGES --filter HL7.MESSAGES --ack all --target HL7MESSAGES --deliver last --replay instant --wait=70s --max-deliver=-1 --max-pending=100 --heartbeat=-1 --flow-control
nats con add HL7 ZIPPED_BATCHES --filter HL7.ZIPPED_BATCHES --ack explicit --pull --deliver all --max-deliver=-1 --sample 100 --max-pending=1 --replay=instant --wait=1s
sleep 100000