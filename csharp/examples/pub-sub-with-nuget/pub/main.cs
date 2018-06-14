using System;
using Deluxe.One.Nats;

namespace ConsoleApp6
{
    class Program
    {
        static void Main(string[] args)
        {
            string serverURL = "nats://localhost:4222", clusterID = "test-cluster", clientID = "pub_client", subject = "foo_subject", message = "foo_message";
            nats.Connect(serverURL, clusterID, clientID);
            nats.Publish(subject, System.Text.Encoding.UTF8.GetBytes(message));
            nats.Close();
        }
    }
}
