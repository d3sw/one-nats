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
        private Thread _thread = null;
        private ManualResetEvent _abort = new ManualResetEvent(false);
        public void Start()
        {
            nats.DefaultPublishRetryDelays = new TimeSpan[] { TimeSpan.FromSeconds(10), TimeSpan.FromSeconds(20), TimeSpan.FromSeconds(30), TimeSpan.FromSeconds(60) };
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
    class Program
    {
        static void Main()
        {
            // init
            var service = new MyService();
            // start
            service.Start();
            // wait for exit
            var ev = new AutoResetEvent(false);
            Console.CancelKeyPress += (sender, e) =>
            {
                service.Stop();
                // return
                e.Cancel = true;
                ev.Set();
            };
            Console.WriteLine("Program started, press ctrl+c to exit...");
            ev.WaitOne();
        }
    }
}
