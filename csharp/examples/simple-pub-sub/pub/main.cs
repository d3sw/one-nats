using STAN.Client;
using System;
using System.Threading;
using Deluxe.One.Nats;

namespace ConsoleApp1
{
    class Program
    {
        static void Main(string[] args)
        {
            string serverURL = "nats://localhost:4222", clusterID = "test-cluster", clientID = "pub_client", subject = "foo_subject";
            nats.Connect(serverURL, clusterID, clientID);
            Console.WriteLine("nats: connected serverURL='{0}', clusterID='{1}', clientID='{2}'", serverURL, clusterID, clientID);
            // publish message
            var ev = new AutoResetEvent(false);
            var seq = 0;
            var thread = new Thread(obj =>
            {
                do
                {
                    var msg = string.Format("message [#{0}]", ++seq);
                    try
                    {
                        nats.Publish(subject, System.Text.Encoding.UTF8.GetBytes(msg));
                    }
                    catch
                    {
                    }
                } while (!ev.WaitOne(TimeSpan.FromSeconds(5)));
            });
            thread.Start();
            // wait for exit
            Console.CancelKeyPress += (sender, e)=>
            {
                e.Cancel = true;
                ev.Set();
            };
            Console.WriteLine("program: ctrl+c to exit...");
            //Console.ReadLine();
            //ev.Set();
            thread.Join();
            nats.Close();
        }
    }
}
