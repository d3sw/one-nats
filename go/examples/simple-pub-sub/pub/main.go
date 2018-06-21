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
	"fmt"
	"os"
	"os/signal"
	"time"

	nats "github.com/d3sw/one-nats/go"
	log "github.com/sirupsen/logrus"
)

// MyService ...
type MyService struct {
	threadAbort, threadDone chan bool
}

// Startup ...
func (m *MyService) Startup() error {
	// init
	nats.Connect("nats://localhost:4222", "test-cluster", "pub_client")
	m.threadAbort = make(chan bool)
	m.threadDone = make(chan bool)
	// run
	go m.run()
	return nil
}

func (m *MyService) waitOne(abort chan bool, delay time.Duration) bool {
	select {
	case <-time.After(time.Second * 5):
		return false
	case <-m.threadAbort:
		return true
	}
}

func (m *MyService) run() {
	seq := 0
	for !m.waitOne(m.threadAbort, time.Second*5) {
		seq++
		msg := fmt.Sprintf("Message [#%d]", seq)
		nats.Publish("foo_subject", []byte(msg))
	}
	m.threadDone <- true
}

// Shutdown ...
func (m *MyService) Shutdown() error {
	// abort thread and wait
	m.threadAbort <- true
	<-m.threadDone
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
			myService.Shutdown()
			// all done
			done <- true
		}
	}()
	<-done
}
