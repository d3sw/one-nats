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
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/go-nats-streaming"
	log "github.com/sirupsen/logrus"
)

// Version is the one-nats client
const Version = "0.1.9"

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
	// Connect connects to nat server
	Connect(serverURL, clusterID, serverID string) error
	// Close closes connect to nats server
	Close() error
	// QueueSubscribe subscribes to a group channed
	QueueSubscribe(subject, queue, durable string, cb stan.MsgHandler, opts ...stan.SubscriptionOption) (SubToken, error)
	// Subscribe subscribes to a channel
	Subscribe(subject, durable string, cb stan.MsgHandler, opts ...stan.SubscriptionOption) (SubToken, error)
	// Unsubscribe unscribe to a subscription
	Unsubscribe(subToken SubToken) error
	// Closesubscribe close a subscription
	Closesubscribe(subToken SubToken) error
	// Publish publishs a message
	Publish(subject string, data []byte, delays ...time.Duration) error
}

var (
	// DefaultMaxInflight default subscriber max inflight messages
	DefaultMaxInflight = 1
	// DefaultReconnectDelay default reconnect delay
	DefaultReconnectDelay = time.Second * 15
	// DefaultPublishRetryDelays default publish retry delays
	DefaultPublishRetryDelays = []time.Duration{time.Second * 0, time.Second * 10, time.Second * 20}
	// DefaultSubscribeAckWait default subscriber Ack wait time
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
func Connect(serverURL, clusterID, serviceID string) error {
	return getDefaultNats().Connect(serverURL, clusterID, serviceID)
}

// Close ...
func Close() error {
	err := getDefaultNats().Close()
	setDefaultNats(nil)
	return err
}

// QueueSubscribe ...
func QueueSubscribe(subject, queue, durable string, cb stan.MsgHandler, opts ...stan.SubscriptionOption) (SubToken, error) {
	return getDefaultNats().QueueSubscribe(subject, queue, durable, cb, opts...)
}

// Subscribe ...
func Subscribe(subject, durable string, cb stan.MsgHandler, opts ...stan.SubscriptionOption) (SubToken, error) {
	return getDefaultNats().Subscribe(subject, durable, cb, opts...)
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
func Publish(subj string, data []byte, delays ...time.Duration) error {
	return getDefaultNats().Publish(subj, data, delays...)
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
	serviceID      string                 // serviceID
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
	opts    []stan.SubscriptionOption
	cb      stan.MsgHandler
	sub     stan.Subscription
}

func (m *Nats) getConnLogger() *log.Entry {
	return log.WithFields(log.Fields{
		"natsURL":        m.serverURL,
		"clusterID":      m.clusterID,
		"serviceID":      m.serviceID,
		"clientID":       m.clientID,
		"onenatsVersion": Version,
		"lang":           "go",
	})
}

// Connect ...
func (m *Nats) Connect(serverURL, clusterID, serviceID string) error {
	// reset values
	if serverURL == "" {
		serverURL = "nats://localhost:4222"
	}
	if clusterID == "" {
		clusterID = "test-cluster"
	}
	if serviceID == "" {
		serviceID = "service"
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
	m.serviceID = m.getServiceID(serviceID)
	m.reconnectAbort = make(chan bool)
	m.reconnectTimer = time.NewTimer(m.ReconnectDelay)
	m.subs = make(map[SubToken]subRecord)
	// now connect to nats
	err := m.reconnect()
	if err != nil {
		m.getConnLogger().WithField("error", err).Warnf("nats connection failed at connect, retry in %s...", m.ReconnectDelay)
	}
	// from nats streaming server 0.10.0, it will support client pinging feature, then you don't need this background trhead
	go m.reconnectServer()
	// return
	return nil
}

func (m *Nats) internalClose() error {
	m.Lock()
	defer m.Unlock()
	// logger
	logger := m.getConnLogger()
	// close subscription
	for _, item := range m.subs {
		if item.sub != nil {
			item.sub.Close()
			item.sub = nil
			logger.WithFields(log.Fields{"subject": item.subject, "queue": item.queue, "durable": item.durable}).Info("nats subscription closed")
		}
	}
	// close that nats connection
	if m.conn != nil {
		// close
		m.conn.Close()
		m.conn = nil
		logger.Info("nats connection closed")
	}
	// return
	return nil
}

func (m *Nats) getServiceID(serviceID string) string {
	var re = regexp.MustCompile(`(^.+?-).{8}-.{4}-.{4}-.{4}-.{12}$`)
	ret := re.ReplaceAllString(serviceID, `$1`)
	ret = strings.Trim(ret, "-")
	ret = strings.TrimSpace(ret)
	if ret == "" {
		ret = serviceID
	}
	// return
	return ret
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
	// create a new clientID
	m.clientID = fmt.Sprintf("%s-%s", m.serviceID, uuid.Must(uuid.NewRandom()).String())
	logger := m.getConnLogger()
	// now connect to nats
	conn, err := DefaultStan.StanConnect(m.clusterID, m.clientID, stan.NatsURL(m.serverURL),
		stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
			logger.WithField("reason", reason).Error("nats connection lost")
			// close the connection
			m.internalClose()
		}))
	if err != nil {
		m.resetReconnectTimer()
		return err
	}
	m.conn = conn
	// now create a new object
	logger.Info("nats connection completed")
	// now setup the subscription
	for _, item := range m.subs {
		item.sub, err = m.internalSubscribe(item.subject, item.queue, item.durable, item.cb, item.opts...)
	}
	// return
	return nil
}

func (m *Nats) resetReconnectTimer() {
	m.reconnectTimer.Reset(m.ReconnectDelay)
}

func (m *Nats) reconnectServer() {
	retries := 0
	logger := m.getConnLogger()
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

func (m *Nats) internalSubscribe(subject, queue, durable string, cb stan.MsgHandler, opts ...stan.SubscriptionOption) (stan.Subscription, error) {
	logger := m.getConnLogger().WithFields(log.Fields{"subject": subject, "queue": queue})
	// check connection first
	if err := m.reconnect(); err != nil {
		logger.WithField("error", err).Warn("nats subscription failed at reconnect")
		return nil, err
	}
	// set options
	var setSubOpts = func(o *stan.SubscriptionOptions) error {
		// setup default options
		o.DurableName = durable        // for durable message
		o.MaxInflight = m.MaxInflight  // for processing one message at a time, to avoid timeout
		o.AckWait = m.SubscribeAckWait // for subscribe ack wait time
		// overwrite with new options
		for _, opt := range opts {
			opt(o)
		}
		logger = logger.WithFields(log.Fields{
			"durable":     o.DurableName,
			"maxInflight": o.MaxInflight,
			"ackWait":     o.AckWait,
			"manualAcks":  o.ManualAcks,
		})
		return nil
	}
	// now sub
	sub, err := stan.Subscription(nil), error(nil)
	if queue == "" {
		sub, err = m.conn.Subscribe(subject, cb, setSubOpts)
	} else {
		sub, err = m.conn.QueueSubscribe(subject, queue, cb, setSubOpts)
	}
	if err != nil {
		logger.WithField("error", err).Warn("nats subscription failed at subscribe")
		return sub, err
	}
	logger.Info("nats subscription completed")
	// return
	return sub, err
}

// Subscribe ...
func (m *Nats) Subscribe(subject, durable string, cb stan.MsgHandler, opts ...stan.SubscriptionOption) (SubToken, error) {
	return QueueSubscribe(subject, "", durable, cb, opts...)
}

// QueueSubscribe ...
func (m *Nats) QueueSubscribe(subject, queue, durable string, cb stan.MsgHandler, opts ...stan.SubscriptionOption) (SubToken, error) {
	// subscribe
	sub, err := m.internalSubscribe(subject, queue, durable, cb, opts...)
	// keep a copy of subscription
	subToken := SubToken(uuid.New().String())
	m.subs[subToken] = subRecord{
		subject: subject,
		queue:   queue,
		durable: durable,
		opts:    opts,
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
		logger.WithFields(log.Fields{"error": err}).Warn("nats unsubscription failed")
	} else {
		logger.Info("nats unsubscription completed")
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
		logger.WithFields(log.Fields{"error": err}).Warn("nats closesubscription failed")
	} else {
		logger.Info("nats closesubscription completed")
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
func (m *Nats) Publish(subj string, data []byte, delays ...time.Duration) error {
	// check
	if subj == "" {
		return errors.New("invalid parameter. subject is empty")
	}
	// logger
	logger := log.WithFields(log.Fields{"subject": subj, "data": string(data)})
	// check delays
	if delays == nil {
		delays = m.PublishRetryDelays
	}
	// publish now with retries
	err := m.retryWithDelays(logger, "nats publish", delays, func() error {
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
