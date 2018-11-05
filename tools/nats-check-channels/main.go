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
	"time"

	"github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/go-nats-streaming/pb"
	log "github.com/sirupsen/logrus"
)

type Subscription struct {
	ClientID     string `json:"client_id"`
	Inbox        string `json:"inbox"`
	AckInbox     string `json:"ack_inbox"`
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

type Channels struct {
	ClusterID string    `json:"cluster_id"`
	ServerID  string    `json:"server_id"`
	Now       time.Time `json:"now"`
	Offset    int       `json:"offset"`
	Limit     int       `json:"limit"`
	Count     int       `json:"count"`
	Total     int       `json:"total"`
	Names     []string  `json:"names"`
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
	log.SetFormatter(&log.TextFormatter{DisableColors: true, QuoteEmptyFields: true})
	var clusterID string
	var server string
	// init
	flag.StringVar(&server, "s", "nats.service.owf-live", "The nats server URLs (separated by comma)")
	flag.StringVar(&server, "server", "nats.service.owf-live", "The nats server URLs (separated by comma)")
	flag.StringVar(&clusterID, "c", "events-streaming", "The NATS Streaming cluster ID")
	flag.StringVar(&clusterID, "cluster", "events-streaming", "The NATS Streaming cluster ID")
	flag.Parse()
	// log info
	log.WithFields(log.Fields{"host": server, "cluster": clusterID}).Info("Checking nats streaming channels")

	channelIDs, _ := getChannelIDs(server)
	// now run
	for _, channelID := range channelIDs {
		checkChannel(server, clusterID, channelID)
	}
}

func checkChannel(server, clusterID, channelID string) {
	lastSents := map[string]int{}
	channel, _ := getChannel(server, clusterID, channelID)
	for _, sub := range channel.Subscriptions {
		if sub.IsOffline {
			continue
		}
		last := lastSents[sub.QueueName]
		if last < sub.LastSent {
			lastSents[sub.QueueName] = sub.LastSent
		}
	}
	for name, last := range lastSents {
		if last != 0 && channel.LastSeq > last+2 {
			fmt.Println(channel.Name, ": last_seq:", channel.LastSeq, "queue:", name, ",last_sent:", last)
		} else {
			// log.Info("channel is OK. name: ", channel.Name)
		}
	}
}

func getChannelIDs(server string) ([]string, error) {
	url := fmt.Sprintf("http://%s:8222/streaming/channelsz", server)
	var items Channels
	err := getJSON(url, &items)
	if err != nil {
		return nil, err
	}
	return items.Names, nil
}

func getChannel(server string, cluster string, channel string) (*Channel, error) {
	url := fmt.Sprintf("http://%s:8222/streaming/channelsz?channel=%s&subs=1", server, channel)
	var item Channel
	err := getJSON(url, &item)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func getJSON(url string, object interface{}) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(object)
}

func closeClient(server string, clusterID string, clientID string) error {
	// run now
	serverURL := fmt.Sprintf("nats://%s:4222", server)
	conn, err := stan.Connect(clusterID, "one-nats-cleanup-tool", stan.NatsURL(serverURL))
	if err != nil {
		return err
	}
	defer conn.Close()
	// force to close a client
	req := &pb.CloseRequest{ClientID: clientID}
	b, _ := req.Marshal()
	reply, err := conn.NatsConn().Request("_STAN.close.events-streaming", b, time.Second*60)
	res := &pb.CloseResponse{}
	err = res.Unmarshal(reply.Data)
	// log
	fmt.Printf("force close clientID: %s, error: %s\n", clientID, res.Error)
	// return
	return nil
}
