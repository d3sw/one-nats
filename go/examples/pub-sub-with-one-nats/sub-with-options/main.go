/*************************************************************************

*

 * COPYRIGHT 2018 Deluxe Entertainment Services Group Inc. and its subsidiaries (“Deluxe”)

*  All Rights Reserved.

*

 * NOTICE:  All information contained herein is, and remains

* the property of Deluxe and its suppliers,

* if any.  The intellectual and technical concepts contained

* herein are proprietary to Deluxe and its suppliers and may be covered

 * by U.S. and Foreign Patents, patents in process, and are protected

 * by trade secret or copyright law.   Dissemination of this information or

 * reproduction of this material is strictly forbidden unless prior written

 * permission is obtained from Deluxe.

*/
package main

import (
	"os"
	"os/signal"

	nats "github.com/d3sw/one-nats/go"
	"github.com/nats-io/go-nats-streaming"
	log "github.com/sirupsen/logrus"
)

// MyService ...
type MyService struct {
}

// Startup ...
func (m *MyService) Startup() error {
	// init
	nats.Connect("nats://localhost:4222", "test-cluster", "sub_client")
	nats.QueueSubscribe("foo_subject", "foo_queue", "foo_durable", m.onReceived,
		stan.MaxInflight(11),
		stan.DurableName("new_durable_name"),
		stan.SetManualAckMode())
	return nil
}

func (m *MyService) onReceived(msg *stan.Msg) {
	log.WithFields(log.Fields{"sequence": msg.Sequence, "redelivered": msg.Redelivered, "message": msg.String()}).Info("received message")
	msg.Ack()
}

// Shutdown ...
func (m *MyService) Shutdown() error {
	// close the nats
	nats.Close()
	return nil
}

func main() {
	// init
	log.SetFormatter(&log.TextFormatter{DisableColors: true, QuoteEmptyFields: true})
	myService := MyService{}
	// connect
	myService.Startup()

	// Wait for a SIGINT (perhaps triggered by user with CTRL-C)
	// Run cleanup when signal is received
	exit, done := make(chan os.Signal, 1), make(chan bool)
	signal.Notify(exit, os.Interrupt)
	go func() {
		for range exit {
			// close nats
			nats.Close()
			// all done
			done <- true
		}
	}()
	<-done
}
