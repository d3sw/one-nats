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
using STAN.Client;
using System;
using System.Threading;
using Deluxe.One.Nats;

namespace ConsoleApp1
{
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
    class Program
    {
        static void Main()
        {
            // init
            var service = new MyService();
            // start
            service.Startup();
            // wait for exit
            var ev = new AutoResetEvent(false);
            Console.CancelKeyPress += (sender, e) =>
            {
                service.Shutdown();
                // return
                e.Cancel = true;
                ev.Set();
            };
            Console.WriteLine("Program started, press ctrl+c to exit...");
            ev.WaitOne();
        }
    }
}