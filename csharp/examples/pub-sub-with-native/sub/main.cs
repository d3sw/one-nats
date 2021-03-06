﻿/*************************************************************************

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

namespace sub
{
    class Program
    {
        static void Main()
        {
            string serverURL = "nats://localhost:4222", clusterID = "test-cluster", clientID = "client_sub", subject = "foo_subject", queue = "foo_queue", durable = "foo_durable";
            StanOptions cOpts = StanOptions.GetDefaultOptions();
            cOpts.NatsURL = serverURL;
            using (var c = new StanConnectionFactory().CreateConnection(clusterID, clientID, cOpts))
            {
                Console.WriteLine("nats: connected serverURL='{0}', clusterID='{1}', clientID='{2}'", serverURL, clusterID, clientID);
                StanSubscriptionOptions sOpts = StanSubscriptionOptions.GetDefaultOptions();
                sOpts.DurableName = durable;
                sOpts.MaxInflight = 1;
                using (var s = c.Subscribe(subject, queue, sOpts, (sender, args) =>
                {
                    Console.WriteLine("Received seq # {0}: {1}", args.Message.Sequence, System.Text.Encoding.UTF8.GetString(args.Message.Data));
                }))
                {
                    Console.WriteLine("nats: subscribed subject='{0}', queue='{1}', durable='{2}'", subject, queue, durable);
                    Console.WriteLine("program: press <enter> to exit...");
                    Console.ReadLine();
                }
            }
        }
    }
}
