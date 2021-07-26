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
	"log"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func setup() {
}

func shutdown() {
}

func TestBatchLimit(t *testing.T) {

}

func TestBatchTimeout(t *testing.T) {

}

func TestZip(t *testing.T) {
	timeBetweenMsgs, _ := time.ParseDuration("1s")
	testMsgs := make([]nats.Msg, 0)
	testMsg1 := nats.NewMsg("test_stream.subject")
	testMsg1.Sub = new(nats.Subscription)
	testMsg1.Reply = "$JS.ACK.3.4.5.6.7.8.9"
	md, err := testMsg1.Metadata()
	if nil != err {
		log.Print(err)
	}
	md.Timestamp = time.Now()
	time.Sleep(timeBetweenMsgs)
	zippedMsgs := zipBatch(testMsgs)
	log.Print(len(zippedMsgs))
}

func TestConfig(t *testing.T) {
	const nats_url = "nats://test-nats:4222"
	const nats_stream_name = "test_nats_stream"
	const nats_incoming_subject = "test_nats_stream.in"
	const nats_outgoing_subject = "test_nats_stream.out"
	const nats_queue_name = "test_queue"
	const msg_batch_timeout = "1111s"
	timeout, _ := time.ParseDuration(msg_batch_timeout)
	const msg_batch_size = "1000000"
	const msg_batch_size_uint = uint64(1000000)
	const nats_durable_name = "so_durable"
	os.Setenv("NATS_URL", nats_url)
	os.Setenv("NATS_STREAM_NAME", nats_stream_name)
	os.Setenv("NATS_INCOMING_SUBJECT_NAME", nats_incoming_subject)
	os.Setenv("NATS_OUTGOING_SUBJECT_NAME", nats_outgoing_subject)
	os.Setenv("NATS_QUEUE_NAME", nats_queue_name)
	os.Setenv("MSG_BATCH_TIMEOUT", msg_batch_timeout)
	os.Setenv("MSG_BATCH_SIZE", msg_batch_size)
	os.Setenv("NATS_DURABLE_NAME", nats_durable_name)

	config := readConfigFromEnv()

	if config.natsUrl != nats_url {
		t.Errorf("config.natsUrl = %q; expected %q", config.natsUrl, nats_url)
	}
	if config.streamName != nats_stream_name {
		t.Errorf("config.streamName = %q; expected %q", config.streamName, nats_stream_name)
	}
	if config.subjectNameIn != nats_incoming_subject {
		t.Errorf("config.subjectNameIn = %q; expected %q", config.subjectNameIn, nats_incoming_subject)
	}
	if config.subjectNameOut != nats_outgoing_subject {
		t.Errorf("config.subjectNameIn = %q; expected %q", config.subjectNameIn, nats_outgoing_subject)
	}
	if config.queueName != nats_queue_name {
		t.Errorf("config.queueName = %q; expected %q", config.queueName, nats_queue_name)
	}
	if config.batchSize != msg_batch_size_uint {
		t.Errorf("config.batchSize = %d; expected %d", config.batchSize, msg_batch_size_uint)
	}
	if config.timeout != timeout {
		t.Errorf("config.timeout = %q; expected %q", config.timeout.String(), timeout.String())
	}
	if config.durableName != nats_durable_name {
		t.Errorf("config.durableName = %q; expected %q", config.durableName, nats_durable_name)
	}
}
