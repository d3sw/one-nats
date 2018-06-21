# one-nats
One-Nats is an imporved nats streaming library, it's a wrapper on the top of the nats streaming client to handle the following three situlations:
1. When nats server is not available when client is connecting to server
1. When nats server is restarted after client subscribed to server
1. When nats server is not available when client is publishing message

# Sequence diagram
## Publisher 
![alt sequence diagram for publisher](docs/one-nats.pub.png)

## Subscriber
![alt sequence diagram for subscriber](docs/one-nats.sub.png)

# Usage
## go
### Publisher
Go publisher example is at [https://github.com/d3sw/one-nats/blob/master/go/examples/simple-pub-sub/pub/main.cs]
```go

// MyService ...
type MyService struct {
	threadAbort, threadDone chan bool
}

// Startup ...
func (m *MyService) Startup() error {
	// init
	nats.Connect("nats://localhost:4222", "test-cluster", "pub_client")
	m.threadAbort = make(chan bool)
	m.threadDone = make(chan bool)
	// run
	go m.run()
	return nil
}

func (m *MyService) waitOne(abort chan bool, delay time.Duration) bool {
	select {
	case <-time.After(time.Second * 5):
		return false
	case <-m.threadAbort:
		return true
	}
}

func (m *MyService) run() {
	seq := 0
	for !m.waitOne(m.threadAbort, time.Second*5) {
		seq++
		msg := fmt.Sprintf("Message [#%d]", seq)
		nats.Publish("foo_subject", []byte(msg))
	}
	m.threadDone <- true
}

// Shutdown ...
func (m *MyService) Shutdown() error {
	// abort thread and wait
	m.threadAbort <- true
	<-m.threadDone
	// close the nats
	nats.Close()
	return nil
}
```
### Subscriber
Go publisher example is at [https://github.com/d3sw/one-nats/blob/master/go/examples/simple-pub-sub/sub/main.cs]
```go

// MyService ...
type MyService struct {
}

// Startup ...
func (m *MyService) Startup() error {
	// init
	nats.Connect("nats://localhost:4222", "test-cluster", "sub_client")
	nats.QueueSubscribe("foo_subject", "foo_queue", "foo_durable", m.onReceived)
	return nil
}

func (m *MyService) onReceived(msg *stan.Msg) {
	log.WithFields(log.Fields{"message": msg.String()}).Info("received message")
}

// Shutdown ...
func (m *MyService) Shutdown() error {
	// close the nats
	nats.Close()
	return nil
}
```

## CSharp
For CSharp project, the nats client project needs to import the one-nats nuget package:
```text
nuget server: http://nuget.service.owf-dev:5000
package ID: Deluxe.One.Nats
```
### Publisher
CSharp publisher example is at [https://github.com/d3sw/one-nats/blob/master/csharp/examples/pub-sub-with-nuget/pub/main.cs]
```csharp
    class MyService
    {
        private Thread _thread = null;
        private ManualResetEvent _abort = new ManualResetEvent(false);
        public void Start()
        {
            // customize the default values
            nats.DefaultPublishRetryDelays = new TimeSpan[] { TimeSpan.FromSeconds(10), TimeSpan.FromSeconds(20), TimeSpan.FromSeconds(30), TimeSpan.FromSeconds(60) };
            // connect
            nats.Connect("nats://localhost:4222", "test-cluster", "pub_client");
            // start the process thread
            _thread = new Thread(onThread);
            _thread.Start();
        }
        private void onThread() {
            var idx = 0;
            while(!_abort.WaitOne(TimeSpan.FromSeconds(5))) {
                var msg = string.Format("message #{0}", ++idx);
                nats.Publish("foo_subject", System.Text.Encoding.UTF8.GetBytes(msg));
            }
        }
        public void Stop()
        {
            _abort.Set();
            _thread.Join();
            nats.Close();
        }
    }
```
### Subscribler
CSharp subsriber example is at [https://github.com/d3sw/one-nats/blob/master/csharp/examples/pub-sub-with-nuget/sub/main.cs]
```csharp
    class MyService
    {
        public void Startup()
        {
            nats.DefaultPublishRetryDelays = new TimeSpan[] { TimeSpan.FromSeconds(10), TimeSpan.FromSeconds(20), TimeSpan.FromSeconds(30), TimeSpan.FromSeconds(60) };
            nats.Connect("nats://localhost:4222", "test-cluster", "sub_client");
            // start the subscriber
            nats.QueueSubscribe("foo_subject", "queue", "durable", onReceived);
        }

        public void Shutdown()
        {
            nats.Close();
        }

        private void onReceived(object sender, StanMsgHandlerArgs args)
        {
            Console.WriteLine("Received seq #{0}: {1}", args.Message.Sequence, System.Text.Encoding.UTF8.GetString(args.Message.Data));
        }
    }
```

## java
