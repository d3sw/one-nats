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

func usage() {
	fmt.Println(`
Usage: one-nats-pub [options] <subject> <message>
	-s, --server, env:NATS_URL			<server url> NATS Streaming server URL(s)
	-c, --clusterID, env:NATS_CLUSTERID	<cluster ID> NATS Streaming cluster ID
	-id, --serverID, env:NATS_SERVICEID	<service ID> NATS Streaming client ID
`)
}

func parseFlags() (URL, clusterID, serviceID string) {
	// flags
	flag.StringVar(&URL, "s", "", "The nats server URLs (separated by comma)")
	flag.StringVar(&URL, "server", "", "The nats server URLs (separated by comma)")
	flag.StringVar(&clusterID, "c", "", "The NATS Streaming cluster ID")
	flag.StringVar(&clusterID, "clusterid", "", "The NATS Streaming cluster ID")
	flag.StringVar(&serviceID, "id", "", "The NATS Streaming client ID to connect with")
	flag.StringVar(&serviceID, "serviceid", "", "The NATS Streaming client ID to connect with")
	flag.Parse()
	// now reset flags with environment variables
	if URL == "" {
		URL = os.Getenv("NATS_URL")
	}
	if clusterID == "" {
		clusterID = os.Getenv("NATS_CLUSTERID")
	}
	if serviceID == "" {
		serviceID = os.Getenv("NATS_SERVICEID")
	}
	// now reset the default value
	if URL == "" {
		URL = stan.DefaultNatsURL
	}
	if clusterID == "" {
		clusterID = "test-cluster"
	}
	if serviceID == "" {
		serviceID = "onenats"
	}
	// return
	return
}

func main() {
	// init
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{DisableColors: true, QuoteEmptyFields: true})
	// flags
	url, clusterID, clientID := parseFlags()
	args := flag.Args()
	// message
	if len(args) < 2 {
		usage()
		os.Exit(-1)
	}
	topic, messages := args[0], args[1:]
	// now connect
	nats.Connect(url, clusterID, clientID)
	for _, msg := range messages {
		nats.Publish(topic, []byte(msg))
	}
	nats.Close()
}
