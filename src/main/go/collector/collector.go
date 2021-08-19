/*******************************************************************************
 *                                                                             *
 * IBM Confidential                                                            *
 *                                                                             *
 * (C) Copyright IBM Corp. 2021                                                *
 *                                                                             *
 * The source code for this program is not published or otherwise              *
 * divested of its trade secrets, irrespective of what has been                *
 * deposited with the U.S. Copyright Office.                                   *
 *******************************************************************************/

package main

import (
	"archive/zip"
	"bytes"
	"fmt"
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
		"Loaded Configuration:\n\tNATS URL: %q\n\tIncoming Subject: %q\n\tOutgoing Subject%q\n\tBatch Size: %d\n\tTimeout: %q",
		c.natsUrl, c.subjectNameIn, c.subjectNameOut, c.batchSize, c.timeout.String())

	nc, err := nats.Connect(c.natsUrl)
	fatalError(err)
	defer nc.Close()
	js, err := nc.JetStream(nats.PublishAsyncMaxPending(256))
	fatalError(err)
	msgBuffer := make(chan nats.Msg, c.batchSize*2)

	stream, err := js.ConsumerInfo("HL7", "MESSAGES")
	fatalError(err)

	if stream.NumAckPending > 0 {
		log.Printf("There are %d pending ACKs for the consumer. We need to wait %q for all pending acks to expire", stream.NumAckPending, stream.Config.AckWait)
		time.Sleep(stream.Config.AckWait)
		log.Printf("Wait over, start to process messages")
	}

	nc.Subscribe(c.subjectNameIn, func(msg *nats.Msg) {
		msgBuffer <- *msg
	})

	batchChannel := make(chan batchAndLastMsg)
	go collectBatches(c, msgBuffer, batchChannel)
	for {
		batchAndMsg := <-batchChannel
		log.Print("sending")
		pubAck, err := js.Publish(c.subjectNameOut, batchAndMsg.batch)
		logError(err)
		if err == nil {
			log.Printf("Published zipped batch to %q, sequence = %d", c.subjectNameOut, pubAck.Sequence)
			err := batchAndMsg.lastMsg.AckSync()
			logError(err)
		}
	}
}

type config struct {
	natsUrl        string
	subjectNameIn  string
	subjectNameOut string
	batchSize      uint64
	timeout        time.Duration
}

func readConfigFromEnv() config {
	c := new(config)
	c.natsUrl = os.Getenv("NATS_URL")
	if c.natsUrl == "" {
		c.natsUrl = nats.DefaultURL
	}
	c.subjectNameIn = os.Getenv("NATS_INCOMING_SUBJECT_NAME")
	if c.subjectNameIn == "" {
		c.subjectNameIn = "HL7.INCOMING"
	}
	c.subjectNameOut = os.Getenv("NATS_OUTGOING_SUBJECT_NAME")
	if c.subjectNameOut == "" {
		c.subjectNameOut = "HL7STR.ENCRYPTED_BATCHES"
	}
	batchSizeEnvVar := os.Getenv("MSG_BATCH_SIZE")
	c.batchSize = uint64(10) // we just keep using 100 as an example
	if batchSizeEnvVar != "" {
		parsedEnvVar, err := strconv.ParseUint(batchSizeEnvVar, 10, 64)
		fatalError(err)
		c.batchSize = parsedEnvVar
	}
	timeoutEnvVar := os.Getenv("MSG_BATCH_TIMEOUT")
	defaultTimeout, err := time.ParseDuration("9s")
	fatalError(err)
	c.timeout = defaultTimeout
	if batchSizeEnvVar != "" {
		parsedEnvVar, err := time.ParseDuration(timeoutEnvVar)
		fatalError(err)
		c.timeout = parsedEnvVar
	}
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

func checkStreamExists(subjectName string, js nats.JetStreamContext) error {
	streamName := strings.Split(subjectName, ".")[0]
	stream, err := js.StreamInfo(streamName)
	if nil != err {
		return err
	}
	if stream == nil {
		return fmt.Errorf("stream %q does not exist", streamName)
	}
	return nil
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

type batchAndLastMsg struct {
	batch   []byte
	lastMsg nats.Msg
}

func collectBatches(c config, msgBuffer chan nats.Msg, output chan batchAndLastMsg) {
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
			case <-t:
				if len(batch) > 0 {
					log.Printf("Batch timeout exceeded. Proceeding with %d messages", len(batch))
					break out

				}
				log.Printf("No messages in %q, queue is empty", c.timeout.String())
				t = time.After(c.timeout)
			}
		}
		output <- batchAndLastMsg{zipBatch(batch), batch[len(batch)-1]}
	}
}
