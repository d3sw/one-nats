package nats

import (
	"bytes"
	"errors"
	"testing"

	gonats "github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats-streaming"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockConn ...
type MockConn struct {
	mock.Mock
}

func (m *MockConn) Publish(subject string, data []byte) error {
	args := m.Called(subject, data)
	return args.Error(0)
}
func (m *MockConn) PublishAsync(subject string, data []byte, ah stan.AckHandler) (string, error) {
	args := m.Called(subject, data, ah)
	return args.String(0), args.Error(1)
}

func (m *MockConn) Subscribe(subject string, cb stan.MsgHandler, opts ...stan.SubscriptionOption) (stan.Subscription, error) {
	a := []interface{}{subject, cb}
	for _, o := range opts {
		a = append(a, o)
	}
	args := m.Called(a...)
	sub, err := args.Get(0), args.Error(1)
	if sub == nil {
		return nil, err
	}
	return sub.(stan.Subscription), err
}

func (m *MockConn) QueueSubscribe(subject, qgroup string, cb stan.MsgHandler, opts ...stan.SubscriptionOption) (stan.Subscription, error) {
	a := []interface{}{subject, qgroup, cb}
	for _, o := range opts {
		a = append(a, o)
	}
	args := m.Called(a...)
	sub, err := args.Get(0), args.Error(1)
	if sub == nil {
		return nil, err
	}
	return sub.(stan.Subscription), err
}

func (m *MockConn) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockConn) NatsConn() *gonats.Conn {
	args := m.Called()
	return args.Get(0).(*gonats.Conn)
}

// NewMockNats returns a new nats mock object
func NewMockNats() *MockNats {
	ret := &MockNats{}
	DefaultNats = ret
	return ret
}

// MockNats ...
type MockNats struct {
	mock.Mock
}

func (m *MockNats) Connect(serverURL, clusterID, clientID string) error {
	args := m.Called(serverURL, clusterID, clientID)
	return args.Error(0)
}
func (m *MockNats) Close() error {
	args := m.Called()
	return args.Error(0)
}
func (m *MockNats) QueueSubscribe(subject, queue, durable string, cb stan.MsgHandler) (SubToken, error) {
	args := m.Called(subject, queue, durable, cb)
	return args.Get(0).(SubToken), args.Error(1)
}
func (m *MockNats) Subscribe(subject, durable string, cb stan.MsgHandler) (SubToken, error) {
	args := m.Called(subject, durable, cb)
	return args.Get(0).(SubToken), args.Error(1)
}
func (m *MockNats) Unsubscribe(subToken SubToken) error {
	args := m.Called(subToken)
	return args.Error(0)
}
func (m *MockNats) Closesubscribe(subToken SubToken) error {
	args := m.Called(subToken)
	return args.Error(0)
}
func (m *MockNats) Publish(subject string, msg interface{}) error {
	args := m.Called(subject, msg)
	return args.Error(0)
}

// NewMockStan returns a new nats mock object
func NewMockStan() *MockStan {
	ret := &MockStan{}
	DefaultStan = ret
	return ret
}

// MockStan ...
type MockStan struct {
	mock.Mock
}

// StanConnect ...
func (m *MockStan) StanConnect(stanClusterID, clientID string, options ...stan.Option) (stan.Conn, error) {
	a := []interface{}{stanClusterID, clientID}
	for _, o := range options {
		a = append(a, o)
	}
	args := m.Called(a...)
	conn, err := args.Get(0), args.Error(1)
	if conn == nil {
		return nil, err
	}
	return conn.(stan.Conn), err
}

// MockSub ...
type MockSub struct {
	mock.Mock
}

func (m *MockSub) ClearMaxPending() error {
	args := m.Called()
	return args.Error(0)
}
func (m *MockSub) Delivered() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}
func (m *MockSub) Dropped() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}
func (m *MockSub) IsValid() bool {
	args := m.Called()
	return args.Bool(0)
}
func (m *MockSub) MaxPending() (int, int, error) {
	args := m.Called()
	return args.Int(0), args.Int(1), args.Error(2)
}
func (m *MockSub) Pending() (int, int, error) {
	args := m.Called()
	return args.Int(0), args.Int(1), args.Error(2)
}
func (m *MockSub) PendingLimits() (int, int, error) {
	args := m.Called()
	return args.Int(0), args.Int(1), args.Error(2)
}
func (m *MockSub) SetPendingLimits(msgLimit, bytesLimit int) error {
	args := m.Called(msgLimit, bytesLimit)
	return args.Error(0)
}
func (m *MockSub) Unsubscribe() error {
	args := m.Called()
	return args.Error(0)
}
func (m *MockSub) Close() error {
	args := m.Called()
	return args.Error(0)
}

func setupLogs() *bytes.Buffer {
	buf := new(bytes.Buffer)
	log.SetOutput(buf)
	// return
	return buf
}

func Test_Connect_Error(t *testing.T) {
	logs := setupLogs()
	errmsg := "some error"
	// mock
	mockStan := NewMockStan()
	mockStan.On("StanConnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New(errmsg))
	// run
	Connect("", "", "")
	// assert
	mockStan.AssertNumberOfCalls(t, "StanConnect", 1)
	assert.Contains(t, logs.String(), errmsg)
}

func Test_Connect_OK(t *testing.T) {
	logs := setupLogs()
	// mock
	conn := &MockConn{}
	mockStan := NewMockStan()
	mockStan.On("StanConnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(conn, nil)
	// run
	Connect("", "", "")
	// assert
	mockStan.AssertNumberOfCalls(t, "StanConnect", 1)
	assert.Contains(t, logs.String(), "nats connection completed")
}

func Test_Subscribe_Error(t *testing.T) {
	logs := setupLogs()
	errmsg := "some error at subscribe"
	// mock
	DefaultNats = &Nats{}
	mockConn := &MockConn{}
	mockConn.On("Subscribe", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New(errmsg))
	mockConn.On("Close").Return(nil)
	mockStan := NewMockStan()
	mockStan.On("StanConnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockConn, nil)
	// run
	Connect("", "", "")
	Subscribe("foo", "durable", func(msg *stan.Msg) {})
	// assert
	assert.Contains(t, logs.String(), errmsg)
}

func Test_Subscribe_OK(t *testing.T) {
	logs := setupLogs()
	// mock
	mockSub := &MockSub{}
	mockConn := &MockConn{}
	mockConn.On("Subscribe", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockSub, nil)
	mockConn.On("Close").Return(nil)
	mockStan := NewMockStan()
	mockStan.On("StanConnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockConn, nil)
	// run
	Connect("", "", "")
	Subscribe("foo", "durable", func(msg *stan.Msg) {})
	// assert
	assert.Contains(t, logs.String(), "nats subscribe completed")
}

func Test_Publish_Error(t *testing.T) {
	logs := setupLogs()
	errmsg := "some error at publish"
	// mock
	mockConn := &MockConn{}
	mockConn.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(errors.New(errmsg))
	mockConn.On("Close").Return(nil)
	mockStan := NewMockStan()
	mockStan.On("StanConnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockConn, nil)
	// run
	DefaultNats = &Nats{PublishRetryDelays: []int{0}}
	Connect("", "", "")
	err := Publish("subject", "message")
	// assert
	assert.NotNil(t, err, "publish failed")
	assert.Contains(t, logs.String(), errmsg)
}
