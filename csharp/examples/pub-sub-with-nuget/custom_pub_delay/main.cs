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
using System;
using Deluxe.One.Nats;

namespace ConsoleApp6
{
    class Program
    {
        static void Main(string[] args)
        {
            string serverURL = "nats://localhost:4222", clusterID = "test-cluster", clientID = "pub_client", subject = "foo_subject", message = "foo_message";
            nats.Default = new Nats()
            {
                ReconnectDelay = nats.DefaultReconnectDelay,
                PublishRetryDelays = new TimeSpan[] { TimeSpan.FromSeconds(10), TimeSpan.FromSeconds(20), TimeSpan.FromSeconds(30) },
                SubscribeAckWait = nats.DefaultSubscribeAckWait,
            };
            nats.Connect(serverURL, clusterID, clientID);
            nats.Publish(subject, System.Text.Encoding.UTF8.GetBytes(message));
            nats.Close();
        }
    }
}
