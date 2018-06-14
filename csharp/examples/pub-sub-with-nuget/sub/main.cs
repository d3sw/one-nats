using System;
using Deluxe.One.Nats;
using System.Threading;

namespace ConsoleApp6
{
    class Program
    {
        static void Main()
        {
            string serverURL = "nats://localhost:4222", clusterID = "test-cluster", clientID = "sub_client", subject = "foo_subject", queue = "foo_queue", durable = "foo_durable";
            nats.Connect(serverURL, clusterID, clientID);
            nats.QueueSubscribe(subject, queue, durable, (sender, args)=>{
                Console.WriteLine("Received seq #{0}: {1}", args.Message.Sequence, System.Text.Encoding.UTF8.GetString(args.Message.Data));
            });
             // wait for exit
            var ev = new AutoResetEvent(false);
            Console.CancelKeyPress += (sender, e) =>
            {
                nats.Close();
                // return
                e.Cancel = true;
                ev.Set();
            };
            Console.WriteLine("program: ctrl+c to exit...");
            ev.WaitOne();
        }
    }
}
