/*******************************************************************************
 * IBM Watson Imaging Common Application Framework 3.0                         *
 *                                                                             *
 * IBM Confidential                                                            *
 *                                                                             *
 * OCO Source Materials                                                        *
 *                                                                             *
 * (C) Copyright IBM Corp. 2019                                                *
 *                                                                             *
 * The source code for this program is not published or otherwise              *
 * divested of its trade secrets, irrespective of what has been                *
 * deposited with the U.S. Copyright Office.                                   *
 *******************************************************************************/

package main

import (
	"archive/zip"
	"bytes"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	c := readConfigFromEnv()
	log.Printf(
		"Loaded Configuration:\n\tNATS URL: %q\n\tStream Name: %q\n\tIncoming Subject: %q\n\tOutgoing Subject%q\n\tQueue Name: %q\n\tBatch Size: %d\n\tTimeout: %q\n\tDurable Name: %q",
		c.natsUrl, c.streamName, c.subjectNameIn, c.subjectNameOut, c.queueName, c.batchSize, c.timeout.String(), c.durableName)

	nc, err := nats.Connect(c.natsUrl)
	fatalError(err)
	defer nc.Close()
	js, err := nc.JetStream(nats.PublishAsyncMaxPending(256))
	fatalError(err)
	stream, err := js.StreamInfo(c.streamName)
	logError(err)
	if stream == nil {
		log.Printf("creating stream %q and subjects %q and %q", c.streamName, c.subjectNameIn, c.subjectNameOut)
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     c.streamName,
			Subjects: []string{c.subjectNameIn, c.subjectNameOut},
		})
		fatalError(err)
	}
	msgBuffer := make(chan nats.Msg, c.batchSize*2)
	js.Subscribe(c.subjectNameIn, func(msg *nats.Msg) {
		msgBuffer <- *msg
	}, nats.ManualAck(), nats.AckAll(), nats.MaxAckPending(int(c.batchSize)+1), nats.AckWait(c.timeout*2), nats.Durable(c.durableName))
	for {
		batch := make([]nats.Msg, 0)
		t := time.After(c.timeout)
	out:
		for {
			select {
			case m := <-msgBuffer:
				batch = append(batch, m)
				if len(batch) >= int(c.batchSize) {
					log.Printf("%d messages recieved. Proceeding with full batch", len(batch))
					break out
				}
				t = time.After(c.timeout)
			case <-t:
				if len(batch) > 0 {
					log.Printf("Batch timeout exceeded. Proceeding with %d messages", len(batch))
					break out
				}
				log.Printf("No messages in %q, queue is empty", c.timeout.String())
				t = time.After(c.timeout)
			}
		}
		pubAck, err := js.Publish(c.subjectNameOut, zipBatch(batch))
		logError(err)
		if err == nil {
			log.Printf("Published zipped batch to %q, sequence = %d", c.subjectNameOut, pubAck.Sequence)
			err := batch[len(batch)-1].AckSync()
			logError(err)
		}
	}
}

type config struct {
	natsUrl        string
	streamName     string
	subjectNameIn  string
	subjectNameOut string
	queueName      string
	batchSize      uint64
	timeout        time.Duration
	durableName    string
}

func readConfigFromEnv() config {
	c := new(config)
	c.natsUrl = os.Getenv("NATS_URL")
	if c.natsUrl == "" {
		c.natsUrl = nats.DefaultURL
	}
	c.streamName = os.Getenv("NATS_STREAM_NAME")
	if c.streamName == "" {
		c.streamName = "HL7"
	}
	c.subjectNameIn = os.Getenv("NATS_INCOMING_SUBJECT_NAME")
	if c.subjectNameIn == "" {
		c.subjectNameIn = strings.Join([]string{c.streamName, "incoming"}, ".")
	}
	c.subjectNameOut = os.Getenv("NATS_OUTGOING_SUBJECT_NAME")
	if c.subjectNameOut == "" {
		c.subjectNameOut = strings.Join([]string{c.streamName, "batched"}, ".")
	}
	c.queueName = os.Getenv("NATS_QUEUE_NAME")
	batchSizeEnvVar := os.Getenv("HL7_BATCH_SIZE")
	c.batchSize = uint64(100) // we just keep using 100 as an example
	if batchSizeEnvVar != "" {
		parsedEnvVar, err := strconv.ParseUint(batchSizeEnvVar, 10, 64)
		fatalError(err)
		c.batchSize = parsedEnvVar
	}
	timeoutEnvVar := os.Getenv("HL7_BATCH_TIMEOUT")
	defaultTimeout, err := time.ParseDuration("60s")
	fatalError(err)
	c.timeout = defaultTimeout
	if batchSizeEnvVar != "" {
		parsedEnvVar, err := time.ParseDuration(timeoutEnvVar)
		fatalError(err)
		c.timeout = parsedEnvVar
	}
	c.durableName = os.Getenv("NATS_DURABLE_NAME")
	return *c
}

func fatalError(err error) {
	if nil != err {
		log.Fatal(err)
	}
}

func logError(err error) {
	if nil != err {
		log.Print(err)
	}
}

func zipBatch(messages []nats.Msg) []byte {
	b := new(bytes.Buffer)
	w := zip.NewWriter(b)
	for _, message := range messages {
		md, err := message.Metadata() // err should only be returned if m is not a Jetstream message
		logError(err)
		entry, err := w.Create(md.Timestamp.String())
		logError(err)
		_, err = entry.Write(message.Data)
		logError(err)
	}
	err := w.Close()
	logError(err)
	return b.Bytes()
}
