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

	"github.com/nats-io/go-nats-streaming"
	log "github.com/sirupsen/logrus"
)

func main() {
	// init
	log.SetFormatter(&log.TextFormatter{DisableColors: true, QuoteEmptyFields: true})
	// connect
	serverURL, clusterID, clientID := "nats://localhost:4222", "test-cluster", "client_1"
	// serverURL, clusterID, clientID := "nats://events.service.owf-dev:4222", "events-streaming", "client_1"
	subject, queue, durable := "foo_subject", "foo_queue", "foo_durable"
	// connect
	conn, err := stan.Connect(clusterID, clientID, stan.NatsURL(serverURL), stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
		log.WithFields(log.Fields{"reason": reason}).Error("nats connection lost")
	}))
	if err != nil {
		log.Panic(err)
	}
	log.Info("nats conn completed")
	// subscribe
	sub, err := conn.QueueSubscribe(subject, queue, func(m *stan.Msg) {
		log.WithFields(log.Fields{"message": m.String()}).Info("Received message")
		// m.Ack()
	}, stan.DurableName(durable))
	if err != nil {
		log.Panic(err)
	}
	log.Info("nats subscription completed")
	// Wait for a SIGINT (perhaps triggered by user with CTRL-C)
	// Run cleanup when signal is received
	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for range signalChan {
			// close sub
			sub.Close()
			log.Info("nats subscription closed")
			// close sub
			conn.Close()
			log.Info("nats conn closed")
			// all done
			cleanupDone <- true
		}
	}()
	<-cleanupDone
}
