﻿using STAN.Client;
using System;
using System.Threading;

namespace ConsoleApp4
{
    class Program
    {
        static void Main(string[] args)
        {
            string serverURL = "nats://localhost:4222", clusterID = "test-cluster", clientID = "pub_client", subject = "foo_subject";
            StanOptions cOpts = StanOptions.GetDefaultOptions();
            cOpts.NatsURL = serverURL;
            using (var c = new StanConnectionFactory().CreateConnection(clusterID, clientID, cOpts))
            {
                Console.WriteLine("nats: connected serverURL='{0}', clusterID='{1}', clientID='{2}'", serverURL, clusterID, clientID);
                // publish message
                var ev = new AutoResetEvent(false);
                var seq = 0;
                var thread = new Thread(obj =>
                {
                    do
                    {
                        var msg = string.Format("message [#{0}]", ++seq);
                        c.Publish(subject, System.Text.Encoding.UTF8.GetBytes(msg));
                        Console.WriteLine("nats: published subject='{0}', message='{1}'", subject, msg);
                    } while (!ev.WaitOne(TimeSpan.FromSeconds(5)));
                });
                thread.Start();
                // wait for exit
                Console.WriteLine("program: press <enter> to exit...");
                Console.ReadLine();
                ev.Set();
                thread.Join();
            }
        }
    }
}
