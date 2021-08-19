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
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func getTestConfig() config {
	timeout, _ := time.ParseDuration("2s")
	return config{
		"nats://nats-main:422",
		"test_stream.subject",
		"test_stream.subject",
		uint64(10),
		timeout,
	}
}

func getTestMsgs(howMany int) []nats.Msg {
	initialTime := time.Date(2020, 12, 20, 0, 30, 0, 0, time.Now().UTC().Location())
	timeBetweenMsgs, _ := time.ParseDuration("1s")

	testMsgs := make([]nats.Msg, 0)

	for i := 1; i <= howMany; i++ {
		testMsg := nats.NewMsg("test_stream.subject")
		testMsg.Sub = new(nats.Subscription)
		testMsg.Reply = "$JS.ACK.3.4.5.6.7.8.9"
		testMsg.Data = []byte(fmt.Sprintf("Test Message %d", i))
		md, err := testMsg.Metadata()
		if nil != err {
			log.Print(err)
		}
		md.Timestamp = initialTime.Add(timeBetweenMsgs * time.Duration(i) * time.Second)
		testMsgs = append(testMsgs, *testMsg)
	}

	return testMsgs
}

func TestBatching(t *testing.T) {
	c := getTestConfig()
	msgs := getTestMsgs(int(c.batchSize) + 1)
	msgBuffer := make(chan nats.Msg, c.batchSize*2)
	batchChannel := make(chan batchAndLastMsg)
	go collectBatches(c, msgBuffer, batchChannel)
	for _, m := range msgs {
		msgBuffer <- m
	}
	timer := time.AfterFunc(c.timeout*10, func() {
		t.Errorf("batching should've timed out in %q, but took more than %q", c.timeout.String(), (c.timeout * 10).String())
	}) //Don't wait too long!
	for i := 1; i < 3; i++ {
		batch := <-batchChannel
		if i == 1 {
			expectedData := fmt.Sprintf("Test Message %d", c.batchSize)
			if string(batch.lastMsg.Data) != expectedData {
				t.Errorf("last message was %q; expected %q", string(batch.lastMsg.Data), expectedData)
			}
		}
		if i == 2 {
			timer.Stop() // we did it!
		}
	}
	// Now we just make sure we don't get an extra batch
	timer2 := time.After(c.timeout * 2)
out:
	for {
		select {
		case batch := <-batchChannel:
			t.Logf("Extra batch, last msg was %q", string(batch.lastMsg.Data))
			t.Error("Recieved 3 batches; expected 2")
		case <-timer2:
			break out
		}
	}
}

func TestZip(t *testing.T) {
	testMsgs := getTestMsgs(3)
	zippedMsgs := zipBatch(testMsgs)
	if len(zippedMsgs) == 0 {
		t.Error("zipped batch was empty")
	}
	zipReader, err := zip.NewReader(bytes.NewReader(zippedMsgs), int64(len(zippedMsgs)))
	if err != nil {
		t.Errorf("error when opening archive: %q", err)
	}
	if len(zipReader.File) != 3 {
		t.Errorf("unzipped %d messages, expected 3", len(zipReader.File))
	}
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
	os.Setenv("NATS_INCOMING_SUBJECT_NAME", nats_incoming_subject)
	os.Setenv("NATS_OUTGOING_SUBJECT_NAME", nats_outgoing_subject)
	os.Setenv("MSG_BATCH_TIMEOUT", msg_batch_timeout)
	os.Setenv("MSG_BATCH_SIZE", msg_batch_size)

	config := readConfigFromEnv()

	if config.natsUrl != nats_url {
		t.Errorf("config.natsUrl = %q; expected %q", config.natsUrl, nats_url)
	}
	if config.subjectNameIn != nats_incoming_subject {
		t.Errorf("config.subjectNameIn = %q; expected %q", config.subjectNameIn, nats_incoming_subject)
	}
	if config.subjectNameOut != nats_outgoing_subject {
		t.Errorf("config.subjectNameIn = %q; expected %q", config.subjectNameIn, nats_outgoing_subject)
	}
	if config.batchSize != msg_batch_size_uint {
		t.Errorf("config.batchSize = %d; expected %d", config.batchSize, msg_batch_size_uint)
	}
	if config.timeout != timeout {
		t.Errorf("config.timeout = %q; expected %q", config.timeout.String(), timeout.String())
	}
}
