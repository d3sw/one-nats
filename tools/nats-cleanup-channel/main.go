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
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/nats-io/go-nats-streaming"
	log "github.com/sirupsen/logrus"
)

type Subscription struct {
	ClientID     string `json:"client_id"`
	Inbox        string `json:"inbox"`
	AckInbox     string `json:"ack_inbox"`
	DurableName  string `json:"durable_name"`
	QueueName    string `json:"queue_name"`
	IsDurable    bool   `json:"is_durable"`
	IsOffline    bool   `json:"is_offline"`
	MaxInflight  int    `json:"max_inflight"`
	AckWait      int    `json:"ack_wait"`
	LastSent     int    `json:"last_sent"`
	PendingCount int    `json:"pending_count"`
	IsStalled    bool   `json:"is_stalled"`
}

type Channel struct {
	Name          string         `json:"name"`
	Msgs          int            `json:"msgs"`
	Bytes         int            `json:"bytes"`
	FirstSeq      int            `json:"first_seq"`
	LastSeq       int            `json:"last_seq"`
	Subscriptions []Subscription `json:"subscriptions"`
}

var usageStr = `
Usage: nats-cleanup-channel [options] <channel>
Options:
	-s, --server   <url>            NATS Streaming server URL(s)
	-c, --cluster  <cluster name>   NATS Streaming cluster name
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
	var server string
	// params
	flag.StringVar(&server, "s", "nats.service.owf-live", "The nats server URLs (separated by comma)")
	flag.StringVar(&server, "server", "nats.service.owf-live", "The nats server URLs (separated by comma)")
	flag.StringVar(&clusterID, "c", "events-streaming", "The NATS Streaming cluster ID")
	flag.StringVar(&clusterID, "cluster", "events-streaming", "The NATS Streaming cluster ID")
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
		log.Printf("Error: A channel must be specified.")
		usage()
	}
	// init
	log.SetFormatter(&log.TextFormatter{DisableColors: true, QuoteEmptyFields: true})
	log.WithFields(log.Fields{"host": server, "cluster": clusterID}).Info("cleanup nats channels")
	// now run
	for _, channel := range args {
		subs, _ := getSubscriptionsOffline(server, clusterID, channel)
		log.WithFields(log.Fields{"channel": channel, "subs": len(subs)}).Info("cleanup a channel")
		// log
		for _, sub := range subs {
			// if sub.IsOffline {
			err := unscribeOfflineClient(server, clusterID, channel, sub)
			if err != nil {
				log.WithFields(log.Fields{"sub": sub, "error": err}).Error("unscribe failed")
			} else {
				log.WithFields(log.Fields{"clientID": sub.ClientID, "queueName": sub.QueueName}).Info("unscribe completed")
			}
			// }
		}
	}
}

func unscribeOfflineClient(server string, clusterID string, channel string, item Subscription) error {
	// parse queue => durable:queue
	durable, queue, clientID, parts := "", item.QueueName, item.ClientID, strings.Split(item.QueueName, ":")
	if len(parts) >= 2 {
		durable = parts[0]
		queue = parts[1]
	}
	if clientID == "" {
		clientID = "nats-cleanup-channel"
	}
	// run now
	serverURL := fmt.Sprintf("nats://%s:4222", server)
	conn, err := stan.Connect(clusterID, clientID, stan.NatsURL(serverURL))
	if err != nil {
		return err
	}
	// loop now
	sub, err := conn.QueueSubscribe(channel, queue, func(msg *stan.Msg) {}, stan.DurableName(durable))
	if err == nil {
		err = sub.Unsubscribe()
	}
	conn.Close()
	// return
	return err
}

func getJSON(url string, object interface{}) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(object)
}

func getSubscriptions(server string, cluster string, channel string) ([]Subscription, error) {
	url := fmt.Sprintf("http://%s:8222/streaming/channelsz?channel=%s&subs=1", server, channel)
	var item Channel
	err := getJSON(url, &item)
	log.Info("url", url, "err:", err)
	if err != nil {
		return nil, err
	}
	return item.Subscriptions, nil
}

func getSubscriptionsOffline(server string, cluster string, channel string) ([]Subscription, error) {
	var ret []Subscription
	subs, err := getSubscriptions(server, cluster, channel)
	if err != nil {
		return nil, err
	}
	for _, sub := range subs {
		// if sub.IsOffline {
		ret = append(ret, sub)
		// }
	}
	return ret, nil
}
