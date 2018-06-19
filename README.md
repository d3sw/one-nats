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
```go
package main
import nats "github.com/d3sw/one-nats/go"
func main() {
	nats.Connect("nats://localhost:4222", "test-cluster", "pub_client")
	nats.Publish("foo_subject", []byte("foot_message"))
	nats.Close()
}
```
### Subscriber
```go
import (
	"fmt"
	"log"
	nats "github.com/d3sw/one-nats/go"
	"github.com/nats-io/go-nats-streaming"
)
func main() {
	nats.Connect("nats://localhost:4222", "test-cluster", "pub_client")
	nats.QueueSubscribe("foo_subject", "foo_queue", "foo_durable", func(msg *stan.Msg) {
		log.Printf("Received on [%s]: '%s'\n", msg.Subject, msg)
	})
	fmt.Scanln()
	nats.Close()
}
```

## CSharp
For CSharp project, the nats client project needs to import the one-nats nuget package:
```text
nuget server: http://nuget.service.owf-dev:5000
package ID: Deluxe.One.Nats
```
### Publisher
[a CSharp publisher exmaple](csharp/examples/pub)
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
[a CSharp subscribe exmaple](csharp/examples/sub)
```csharp
    class MyService
    {
        public void Start()
        {
            nats.DefaultPublishRetryDelays = new TimeSpan[] { TimeSpan.FromSeconds(10), TimeSpan.FromSeconds(20), TimeSpan.FromSeconds(30), TimeSpan.FromSeconds(60) };
            nats.Connect("nats://localhost:4222", "test-cluster", "sub_client");
            // start the subscriber
            nats.QueueSubscribe("foo_subject", "queue", "durable", (sender, args)=>{
                Console.WriteLine("Received seq #{0}: {1}", args.Message.Sequence, System.Text.Encoding.UTF8.GetString(args.Message.Data));
            });            
        }

        public void Stop()
        {
            nats.Close();
        }
    }
```

## java
