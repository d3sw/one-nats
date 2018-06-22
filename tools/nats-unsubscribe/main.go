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
	"flag"
	"fmt"
	"os"

	nats "github.com/d3sw/one-nats/go"
	"github.com/nats-io/go-nats-streaming"
	log "github.com/sirupsen/logrus"
)

var usageStr = `
Usage: nats-unsubscribe [options] <queue> <subject>
Options:
	-s, --server   <url>            NATS Streaming server URL(s)
	-c, --cluster  <cluster name>   NATS Streaming cluster name
	--qgroup <name>                 Queue group
	--durable <name>                Durable subscriber name
`

// NOTE: Use tls scheme for TLS, e.g. stan-sub -s tls://demo.nats.io:4443 foo
func usage() {
	fmt.Println(usageStr)
	os.Exit(0)
}

func main() {
	var clusterID string
	var clientID string
	var showTime bool
	var startSeq uint64
	var startDelta string
	var deliverAll bool
	var deliverLast bool
	var durable string
	var qgroup string
	var unsubscribe bool
	var serverURL string
	// params
	flag.StringVar(&serverURL, "s", "localhost", "The nats server URLs (separated by comma)")
	flag.StringVar(&serverURL, "server", "localhost", "The nats server URLs (separated by comma)")
	flag.StringVar(&clusterID, "c", "test-cluster", "The NATS Streaming cluster ID")
	flag.StringVar(&clusterID, "cluster", "test-cluster", "The NATS Streaming cluster ID")
	flag.StringVar(&clientID, "id", "", "The NATS Streaming client ID to connect with")
	flag.StringVar(&clientID, "clientid", "", "The NATS Streaming client ID to connect with")
	flag.BoolVar(&showTime, "t", false, "Display timestamps")
	// Subscription options
	flag.Uint64Var(&startSeq, "seq", 0, "Start at sequence no.")
	flag.BoolVar(&deliverAll, "all", false, "Deliver all")
	flag.BoolVar(&deliverLast, "last", false, "Start with last value")
	flag.StringVar(&startDelta, "since", "", "Deliver messages since specified time offset")
	flag.StringVar(&durable, "durable", "", "Durable subscriber name")
	flag.StringVar(&qgroup, "qgroup", "", "Queue group name")
	flag.BoolVar(&unsubscribe, "unsubscribe", false, "Unsubscribe the durable on exit")
	// check usage
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		usage()
	}
	subject := args[0]
	if qgroup == "" || subject == "" {
		usage()
	}
	// init
	log.SetFormatter(&log.TextFormatter{DisableColors: true, QuoteEmptyFields: true})
	// now run
	nats.Connect(serverURL, clusterID, clientID)
	sub, _ := nats.QueueSubscribe(subject, qgroup, durable, func(msg *stan.Msg) {})
	nats.Unsubscribe(sub)
	nats.Close()
}
