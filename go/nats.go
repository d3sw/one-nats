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
package nats

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/go-nats-streaming"
	log "github.com/sirupsen/logrus"
)

// SubToken subscription token
type SubToken string

// IStan inteface
type IStan interface {
	StanConnect(stanClusterID, clientID string, options ...stan.Option) (stan.Conn, error)
}

// Stan ...
type Stan struct {
}

// DefaultStan default connect object
var DefaultStan IStan = &Stan{}

// StanConnect ...
func StanConnect(stanClusterID, clientID string, options ...stan.Option) (stan.Conn, error) {
	return DefaultStan.StanConnect(stanClusterID, clientID, options...)
}

// StanConnect ...
func (m *Stan) StanConnect(stanClusterID, clientID string, options ...stan.Option) (stan.Conn, error) {
	return stan.Connect(stanClusterID, clientID, options...)
}

// INats nats interface
type INats interface {
	Connect(serverURL, clusterID, clientID string) error
	Close() error
	QueueSubscribe(subject, queue, durable string, cb stan.MsgHandler) (SubToken, error)
	Subscribe(subject, durable string, cb stan.MsgHandler) (SubToken, error)
	Unsubscribe(subToken SubToken) error
	Closesubscribe(subToken SubToken) error
	Publish(subject string, data []byte) error
}

var (
	// DefaultMaxInflight default subscriber max inflight messages
	DefaultMaxInflight = 1
	// DefaultReconnectDelay default reconnect delay
	DefaultReconnectDelay = time.Second * 15
	// DefaultPublishRetryDelays default publish retry delays
	DefaultPublishRetryDelays = []time.Duration{time.Second * 0, time.Second * 10, time.Second * 20}
	// SubscribeAckWait default subscriber Ack wait time
	DefaultSubscribeAckWait = time.Second * 60
)

// DefaultNats default nats object
var (
	muDefaultNats sync.Mutex
	DefaultNats   INats
)

func getDefaultNats() INats {
	if DefaultNats == nil {
		muDefaultNats.Lock()
		defer muDefaultNats.Unlock()
		if DefaultNats == nil {
			DefaultNats = &Nats{
				ReconnectDelay:     DefaultReconnectDelay,
				PublishRetryDelays: DefaultPublishRetryDelays,
				SubscribeAckWait:   DefaultSubscribeAckWait,
				MaxInflight:        DefaultMaxInflight,
			}
		}
	}
	return DefaultNats
}

func setDefaultNats(value INats) {
	muDefaultNats.Lock()
	defer muDefaultNats.Unlock()
	DefaultNats = value
}

// Connect ...
func Connect(serverURL, clusterID, clientID string) error {
	return getDefaultNats().Connect(serverURL, clusterID, clientID)
}

// Close ...
func Close() error {
	err := getDefaultNats().Close()
	setDefaultNats(nil)
	return err
}

// QueueSubscribe ...
func QueueSubscribe(subject, queue, durable string, cb stan.MsgHandler) (SubToken, error) {
	return getDefaultNats().QueueSubscribe(subject, queue, durable, cb)
}

// Subscribe ...
func Subscribe(subject, durable string, cb stan.MsgHandler) (SubToken, error) {
	return getDefaultNats().Subscribe(subject, durable, cb)
}

// Unsubscribe unscribe from server
func Unsubscribe(subToken SubToken) error {
	return getDefaultNats().Unsubscribe(subToken)
}

// Closesubscribe closes the subscription
func Closesubscribe(subToken SubToken) error {
	return getDefaultNats().Closesubscribe(subToken)
}

// Publish ...
func Publish(subj string, data []byte) error {
	return getDefaultNats().Publish(subj, data)
}

type pubAborts struct {
	sync.Mutex                    // token
	events     map[chan bool]bool // pub abort events
}

// Nats ...
type Nats struct {
	sync.Mutex                            // token
	conn           stan.Conn              // nats connection
	reconnectTimer *time.Timer            // reconnect timer
	reconnectAbort chan bool              // reconnect abort event
	pubAborts      pubAborts              // pub abort events
	serverURL      string                 // server url
	clusterID      string                 // clusterID
	clientID       string                 // clientID
	subs           map[SubToken]subRecord // pending subscription

	ReconnectDelay     time.Duration   // reconnect delay
	PublishRetryDelays []time.Duration // publish retry delay
	SubscribeAckWait   time.Duration   // subscribe ack wait
	MaxInflight        int             // subscriber max in flight messages
}

// subRecord ...
type subRecord struct {
	subject string
	queue   string
	durable string
	cb      stan.MsgHandler
	sub     stan.Subscription
}

// Connect ...
func (m *Nats) Connect(serverURL, clusterID, clientID string) error {
	// reset values
	if serverURL == "" {
		serverURL = "nats://localhost:4222"
	}
	if clusterID == "" {
		clusterID = "test-cluster"
	}
	if clientID == "" {
		clientID = uuid.Must(uuid.NewRandom()).String()
	}
	// reset value
	if m.ReconnectDelay <= 0 {
		m.ReconnectDelay = DefaultReconnectDelay
	}
	if len(m.PublishRetryDelays) == 0 {
		m.PublishRetryDelays = DefaultPublishRetryDelays
	}
	// init
	m.serverURL = serverURL
	m.clusterID = clusterID
	m.clientID = clientID
	m.reconnectAbort = make(chan bool)
	m.reconnectTimer = time.NewTimer(m.ReconnectDelay)
	m.subs = make(map[SubToken]subRecord)
	// now connect to nats
	err := m.reconnect()
	if err != nil {
		log.WithFields(log.Fields{"clusterID": clusterID, "clientID": clientID, "url": serverURL, "error": err}).Warnf("nats connection failed at connect, retry in %s...", m.ReconnectDelay)
	}
	// from nats streaming server 0.10.0, it will support client pinging feature, then you don't need this background trhead
	go m.reconnectServer()
	// return
	return nil
}

func (m *Nats) internalClose() error {
	m.Lock()
	defer m.Unlock()
	// close subscription
	for _, item := range m.subs {
		if item.sub != nil {
			item.sub.Close()
			item.sub = nil
			log.WithFields(log.Fields{"subject": item.subject, "queue": item.queue, "durable": item.durable}).Info("nats subscription closed")
		}
	}
	// close that nats connection
	if m.conn != nil {
		// close
		m.conn.Close()
		m.conn = nil
		log.WithFields(log.Fields{"server": m.serverURL, "clusterID": m.clusterID, "clientID": m.clientID}).Info("nats connection closed")
	}
	// return
	return nil
}

func (m *Nats) reconnect() error {
	if m.conn != nil {
		return nil
	}
	// now lock data
	m.Lock()
	defer m.Unlock()
	// check m.conn exists or not
	if m.conn != nil {
		return nil
	}
	// now connect to nats
	conn, err := DefaultStan.StanConnect(m.clusterID, m.clientID, stan.NatsURL(m.serverURL),
		stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
			log.WithFields(log.Fields{"clusterID": m.clusterID, "clientID": m.clientID, "url": m.serverURL, "reason": reason}).Error("nats connection lost")
			// close the connection
			m.internalClose()
		}))
	if err != nil {
		m.resetReconnectTimer()
		return err
	}
	m.conn = conn
	// now create a new object
	log.WithFields(log.Fields{"clusterID": m.clusterID, "clientID": m.clientID, "url": m.serverURL}).Info("nats connection completed")
	// now setup the subscription
	for _, item := range m.subs {
		item.sub, err = m.internalSubscribe(item.subject, item.queue, item.durable, item.cb)
	}
	// return
	return nil
}

func (m *Nats) resetReconnectTimer() {
	m.reconnectTimer.Reset(m.ReconnectDelay)
}

func (m *Nats) reconnectServer() {
	retries := 0
	logger := log.WithFields(log.Fields{"url": m.serverURL, "clusterID": m.clusterID, "clientID": m.clientID})
	for {
		select {
		case <-m.reconnectTimer.C:
			break
		case <-m.reconnectAbort:
			return
		}
		// check there is subscription or not
		if len(m.subs) > 0 {
			// now ping server
			if m.conn != nil {
				if err := m.conn.Publish("ping", nil); err != nil {
					logger.WithFields(log.Fields{"error": err}).Warnf("nats ping server failed, reconnect now...")
					// close the connection
					m.internalClose()
				}
			}
			// now reconnect
			if err := m.reconnect(); err != nil {
				retries++
				logger.WithFields(log.Fields{"retries": retries, "error": err}).Warnf("nats reconnection failed, retry in %s...", m.ReconnectDelay)
			}
		}
		// reset timer
		m.resetReconnectTimer()
	}
}

func (m *Nats) getSubscriptionOptions(durableName string) []stan.SubscriptionOption {
	ret := []stan.SubscriptionOption{
		stan.DurableName(durableName),    // for durable message
		stan.MaxInflight(m.MaxInflight),  // for processing one message at a time, to avoid timeout
		stan.AckWait(m.SubscribeAckWait), // for subscribe ack wait time
	}
	return ret
}

func (m *Nats) internalSubscribe(subject, queue, durable string, cb stan.MsgHandler) (stan.Subscription, error) {
	logger := log.WithFields(log.Fields{"url": m.serverURL, "clusterID": m.clusterID, "clientID": m.clientID, "subject": subject, "queue": queue, "durable": durable})
	// check connection first
	if err := m.reconnect(); err != nil {
		logger.WithField("error", err).Warn("nats subscribe failed at reconnect")
		return nil, err
	}
	// now sub
	sub, err, opts := stan.Subscription(nil), error(nil), m.getSubscriptionOptions(durable)
	if queue == "" {
		sub, err = m.conn.Subscribe(subject, cb, opts...)
	} else {
		sub, err = m.conn.QueueSubscribe(subject, queue, cb, opts...)
	}
	if err != nil {
		logger.WithField("error", err).Warn("nats subscription failed at subscribe")
		return sub, err
	}
	logger.Info("nats subscribe completed")
	// return
	return sub, err
}

// Subscribe ...
func (m *Nats) Subscribe(subject, durable string, cb stan.MsgHandler) (SubToken, error) {
	return QueueSubscribe(subject, "", durable, cb)
}

// QueueSubscribe ...
func (m *Nats) QueueSubscribe(subject, queue, durable string, cb stan.MsgHandler) (SubToken, error) {
	// subscribe
	sub, err := m.internalSubscribe(subject, queue, durable, cb)
	// keep a copy of subscription
	subToken := SubToken(uuid.New().String())
	m.subs[subToken] = subRecord{
		subject: subject,
		queue:   queue,
		durable: durable,
		cb:      cb,
		sub:     sub,
	}
	// check error
	if err != nil {
		m.internalClose()
		m.reconnect()
	}
	// return
	return subToken, nil
}

func (m *Nats) removeSubscription(subToken SubToken) (*subRecord, error) {
	m.Lock()
	defer m.Unlock()
	// find the item
	item, ok := m.subs[subToken]
	if ok {
		delete(m.subs, subToken)
		return &item, nil
	}
	return nil, fmt.Errorf("nats subscription not found. subToken='%s'", subToken)
}

// Unsubscribe closes the subscription
func (m *Nats) Unsubscribe(subToken SubToken) error {
	// remove from m.sub
	subRec, err := m.removeSubscription(subToken)
	if err != nil {
		return err
	}
	logger := log.WithFields(log.Fields{"subject": subRec.subject, "queue": subRec.queue, "durable": subRec.durable})
	// close the subscription
	if subRec.sub == nil {
		return nil
	}
	err = subRec.sub.Unsubscribe()
	// log
	if err != nil {
		logger.WithFields(log.Fields{"error": err}).Warn("nats unsubscribe failed")
	} else {
		logger.Info("nats unsubscribe completed")
	}
	// return
	return err
}

// Closesubscribe closes the subscription
func (m *Nats) Closesubscribe(subToken SubToken) error {
	subRec, err := m.removeSubscription(subToken)
	if err != nil {
		return err
	}
	logger := log.WithFields(log.Fields{"subject": subRec.subject, "queue": subRec.queue, "durable": subRec.durable})
	// close the subscription
	if subRec.sub == nil {
		return nil
	}
	err = subRec.sub.Close()
	if err != nil {
		logger.WithFields(log.Fields{"error": err}).Warn("nats closesubscribe failed")
	} else {
		logger.Info("nats closesubscribe completed")
	}
	// return
	return err
}

func (m *Nats) internalPublish(subject string, data []byte) error {
	if err := m.reconnect(); err != nil {
		return err
	}
	err := m.conn.Publish(subject, data)
	// if there is an error close the connect
	if err != nil {
		m.internalClose()
	}
	return err
}

func (m *Nats) retryWithDelays(logger *log.Entry, name string, delays []time.Duration, cb func() error) error {
	var err error
	abort := make(chan bool)
	m.pubAborts.Lock()
	if m.pubAborts.events == nil {
		m.pubAborts.events = make(map[chan bool]bool)
	}
	m.pubAborts.events[abort] = true
	m.pubAborts.Unlock()
	defer func() {
		m.pubAborts.Lock()
		delete(m.pubAborts.events, abort)
		m.pubAborts.Unlock()
	}()
	// look now
	isAborted := false
	for idx, sum := 0, len(delays); !isAborted && idx <= sum; idx++ {
		if err = cb(); err == nil {
			break
		}
		// break now
		if idx >= sum {
			break
		}
		// update logger
		logger = logger.WithField("retry", idx+1)
		// delay
		delay := delays[idx]
		logger.WithFields(log.Fields{"error": err, "delay": delay}).Warnf("%s failed, retry in %s...", name, delay)
		// wait now
		select {
		case <-time.After(delay):
		case <-abort:
			isAborted = true
			break
		}
	}
	return err
}

// Publish ...
func (m *Nats) Publish(subj string, data []byte) error {
	// check
	if subj == "" {
		return errors.New("invalid parameter. subject is empty")
	}
	// logger
	logger := log.WithFields(log.Fields{"subject": subj, "data": string(data)})
	// publish now with retries
	err := m.retryWithDelays(logger, "nats publish", m.PublishRetryDelays, func() error {
		return m.internalPublish(subj, data)
	})
	if err != nil {
		logger.WithFields(log.Fields{"error": err}).Warn("nats publish failed")
	} else {
		logger.Info("nats publish completed")
	}
	// return
	return err
}

// Close ...
func (m *Nats) Close() error {
	// abor the publishing
	m.pubAborts.Lock()
	hasPub := false
	if m.pubAborts.events != nil {
		for key := range m.pubAborts.events {
			key <- true
			delete(m.pubAborts.events, key)
			hasPub = true
		}
	}
	m.pubAborts.Unlock()
	if hasPub {
		time.Sleep(100)
	}
	// stop the reconnect thread
	if m.reconnectAbort != nil {
		m.reconnectAbort <- true
		m.reconnectAbort = nil
	}
	// stop the reconnect thread
	m.internalClose()
	// log
	log.Info("nats closed")
	return nil
}
