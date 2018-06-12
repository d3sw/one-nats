// Copyright 2016-2018 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/d3sw/go-owf/nats"
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
			nats.Publish(subject, fmt.Sprintf("Message [#%d]", seq))
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
