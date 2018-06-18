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

func main() {
	// init
	log.SetFormatter(&log.TextFormatter{DisableColors: true, QuoteEmptyFields: true})
	// connect
	serverURL, clusterID, clientID, subject := "nats://localhost:4222", "test-cluster", "client_pub_simple", "foo_subject"
	nats.Connect(serverURL, clusterID, clientID)
	// now publish a message every 10 seconds
	pubAbort, seq, timer := make(chan bool), 0, time.NewTimer(time.Second*5)
	go func() {
		for {
			seq++
			msg := fmt.Sprintf("Message [#%d]", seq)
			nats.Publish(subject, []byte(msg))
			// wait
			select {
			case <-timer.C:
			case <-pubAbort:
				return
			}
			timer.Reset(time.Second * 5)
		}
	}()

	// Wait for a SIGINT (perhaps triggered by user with CTRL-C)
	// Run cleanup when signal is received
	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for range signalChan {
			// close nats
			nats.Close()
			// abort publish
			pubAbort <- true
			// log
			log.Info("program exited")
			// all done
			cleanupDone <- true
		}
	}()
	<-cleanupDone
}
