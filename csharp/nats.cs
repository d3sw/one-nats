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
using System.Collections.Generic;
using System.Text;
using System.Text.RegularExpressions;
using System.Threading;
using Microsoft.Extensions.Logging;
using STAN.Client;

namespace Deluxe.One.Nats
{
    /// <summary>
    /// Nats connection factory
    /// </summary>
    public interface INatsFactory
    {
        /// <summary>
        /// create a new connection
        /// </summary>
        /// <param name="clusterID"></param>
        /// <param name="clientID"></param>
        /// <param name="options"></param>
        /// <returns></returns>
        IStanConnection CreateConnection(string clusterID, string clientID, StanOptions options);
    }

    /// <summary>
    /// INatsFactory implementation
    /// </summary>
    public class NatsFactory : INatsFactory
    {
        /// <summary>
        /// create a new connection
        /// </summary>
        /// <param name="clusterID"></param>
        /// <param name="clientID"></param>
        /// <param name="options"></param>
        /// <returns></returns>
        public IStanConnection CreateConnection(string clusterID, string clientID, StanOptions options)
        {
            return new StanConnectionFactory().CreateConnection(clusterID, clientID, options);
        }
    }

    public class SubRecord
    {
        public string subject;
        public string queue;
        public string durable;
        public EventHandler<StanMsgHandlerArgs> cb;
        public IStanSubscription sub;
    };

    public interface INats
    {
        /// <summary>
        /// connect to nats server
        /// </summary>
        /// <param name="serverURL"></param>
        /// <param name="clusterID"></param>
        /// <param name="serviceID"></param>
        void Connect(string serverURL, string clusterID, string serviceID);
        /// <summary>
        /// close connection to nats
        /// </summary>
        void Close();
        /// <summary>
        /// publish message to nats
        /// </summary>
        /// <param name="subject"></param>
        /// <param name="data"></param>
        /// <param name="delays"></param>
        void Publish(string subject, byte[] data, TimeSpan[] delays = null);
        /// <summary>
        /// publish message to nats
        /// </summary>
        /// <param name="subject"></param>
        /// <param name="msg"></param>
        /// <param name="delays"></param>
        void Publish(string subject, string msg, TimeSpan[] delays = null);
        /// <summary>
        /// subscribe to nats server
        /// </summary>
        /// <param name="subject"></param>
        /// <param name="durable"></param>
        /// <param name="cb"></param>
        /// <returns>guid to subscription</returns>
        string Subscribe(string subject, string durable, EventHandler<StanMsgHandlerArgs> cb);
        /// <summary>
        /// queue subscription to nats server 
        /// </summary>
        /// <param name="subject"></param>
        /// <param name="queue"></param>
        /// <param name="durable"></param>
        /// <param name="cb"></param>
        /// <returns>guid to subscription</returns>
        string QueueSubscribe(string subject, string queue, string durable, EventHandler<StanMsgHandlerArgs> cb);
        /// <summary>
        /// unscribe to the nats server
        /// </summary>
        /// <param name="guid"></param>
        void Unscribe(string guid);
        /// <summary>
        /// close subscription to nats server
        /// </summary>
        /// <param name="guid"></param>
        /// <returns></returns>
        void Closesubscribe(string guid);
    };
    public class Nats : INats
    {
        public int MaxInflight = nats.DefaultMaxInflight;
        public TimeSpan[] PublishRetryDelays = nats.DefaultPublishRetryDelays;
        public TimeSpan ReconnectDelay = nats.DefaultReconnectDelay;
        public TimeSpan SubscribeAckWait = nats.DefaultSubscribeAckWait;
        public ILogger Logger { get; }

        private object _token = new object();
        private STAN.Client.IStanConnection _conn = null;
        private string _serverURL, _clusterID, _clientID, _serviceID;
        private AutoResetEvent _reconnectAbort = new AutoResetEvent(false);
        private ManualResetEvent _publishAbort = new ManualResetEvent(false);
        private Dictionary<string, SubRecord> _subs = new Dictionary<string, SubRecord>();
        private Thread _reconnectThread;


        public Nats(ILogger logger = null)
        {
            Logger = logger ?? nats.DefaultLogger;
        }

        private Exception reconnect()
        {
            if (_conn != null)
                return null;
            // reconnect
            Exception error = null;
            lock (_token)
            {
                if (_conn != null)
                    return null;
                // create a new clientID
                _clientID = string.Format("{0}-{1}", _serviceID, Guid.NewGuid());
                // fields
                var fields = new Dictionary<string, object>{
                    { "clusterID", _clusterID }, {"clientID", _clientID }, {"serverURL", _serverURL},
                };
                // now create a new connect
                var opts = StanOptions.GetDefaultOptions();
                opts.NatsURL = _serverURL;
                try
                {
                    // reset event
                    _publishAbort.Reset();
                    // reconnect
                    var conn = nats.DefaultFactory.CreateConnection(_clusterID, _clientID, opts);
                    if (conn == null)
                        throw new ApplicationException(string.Format("nats connection failed, conn==null"));
                    // save conn
                    _conn = conn;
                    // log info
                    logInfo(fields, "nats connection completed");
                    // resubscribe all
                    foreach (var item in _subs.Values)
                    {
                        IStanSubscription sub = null;
                        internalSubscribe(item.subject, item.queue, item.durable, item.cb, out sub);
                        item.sub = sub;
                    }
                }
                catch (Exception ex)
                {
                    error = ex;
                }
            }
            // return
            return error;
        }
        public void Connect(string serverURL, string clusterID, string serviceID)
        {
            // reset values
            if (string.IsNullOrEmpty(serverURL))
                serverURL = "nats://localhost:4222";
            if (string.IsNullOrEmpty(clusterID))
                clusterID = "test-cluster";
            if (string.IsNullOrEmpty(serviceID))
                serviceID = "service";
            // save settings
            _serverURL = serverURL;
            _clusterID = clusterID;
            _serviceID = getServiceID(serviceID);
            // now connect to nats
            var error = reconnect();
            if (error != null)
            {
                var fields = new Dictionary<string, object>{
                    { "clusterID", _clusterID},
                    { "clientID", _clientID },
                    { "serverURL", _serverURL },
                    { "error", error } };
                logWarn(fields, "nats connection failed at connect, retry at {0}...", DateTime.Now + ReconnectDelay);
            }
            // from nats streaming server 0.10.0, it will support client pinging feature, then you don't need this background trhead
            _reconnectThread = new Thread(reconnectServer);
            _reconnectThread.Start();
        }

        public void Close()
        {
            // abort publish
            _publishAbort.Set();
            // stop the reconnect thread
            if (_reconnectThread != null)
            {
                _reconnectAbort.Set();
                _reconnectThread.Join();
                _reconnectThread = null;
            }
            // close the 
            internalClose();
            logInfo(null, "nats closed");
        }

        private string getServiceID(string serviceID) {
            var ret = Regex.Replace(serviceID, "(^.+?-).{8}-.{4}-.{4}-.{4}-.{12}$", "$1").Trim('-');
            if (string.IsNullOrWhiteSpace(ret))
                ret = serviceID;
            return ret;
        }

        private void internalClose()
        {
            lock (_token)
            {
                // close all subscription
                foreach (var pair in _subs)
                {
                    Exception error = null;
                    var item = pair.Value;
                    if (item.sub != null)
                    {
                        var fields = new Dictionary<string, object> {
                            { "subject", item.subject },
                            { "queue", item.queue },
                            {"durable", item.durable } };
                        try
                        {
                            item.sub.Close();
                        }
                        catch (Exception ex)
                        {
                            error = ex;
                        }
                        item.sub = null;
                        if (error != null)
                        {
                            fields["error"] = error.Message;
                            logError(fields, "nats subscription close failed");
                        }
                        else
                        {
                            logInfo(fields, "nats subscription close completed");
                        }
                    }
                }
                // close the connection
                if (_conn != null)
                {
                    try { _conn.Close(); } catch { }
                    _conn = null;
                    var fields = new Dictionary<string, object> {
                        { "serverURL",_serverURL },
                        { "clusterID", _clusterID },
                        { "clientID ", _clientID } };
                    logInfo(fields, "nats connection closed");
                }
            }
        }

        private Exception internalPublish(string subject, byte[] data)
        {
            // check connection
            var error = reconnect();
            if (error != null)
                return error;
            // publish message
            try
            {
                _conn.Publish(subject, data);
            }
            catch (Exception ex)
            {
                error = ex;
                internalClose();
            }
            return error;
        }

        public void Publish(string subject, string msg, TimeSpan[] delays = null)
        {
            Publish(subject, Encoding.UTF8.GetBytes(msg), delays);
        }
        public void Publish(string subject, byte[] data, TimeSpan[] delays = null)
        {
            // check value
            if (string.IsNullOrWhiteSpace(subject))
                throw new ApplicationException("invalid parameter. subject is empty");
            // init
            Exception error = null;
            Dictionary<string, object> fields;
            // reset value
            if (delays == null)
                delays = PublishRetryDelays;
            // publish with retries
            for (int idx = 0, sum = delays.Length; idx <= sum; idx++)
            {
                error = internalPublish(subject, data);
                if (error == null)
                    break;
                // break now
                if (idx >= sum)
                    break;
                var delay = delays[idx];
                fields = new Dictionary<string, object>{
                    { "error", error },
                    { "delay", delay } };
                logWarn(fields, "nats publish failed, retry in {0}...", delay);
                // now wait on publish abort for delay
                if (_publishAbort.WaitOne(delay))
                    break;
            }
            // check error
            fields = new Dictionary<string, object> {
                    { "subject", subject },
                    { "data", System.Text.Encoding.UTF8.GetString(data) } };
            if (error != null)
            {
                fields["error"] = error;
                logError(fields, "nats publish failed");
                throw error;
            }
            else
            {
                logInfo(fields, "nats publish completed");
            }
        }

        private StanSubscriptionOptions getSubscriptionOptions(string durable)
        {
            var ret = StanSubscriptionOptions.GetDefaultOptions();
            ret.DurableName = durable;
            ret.MaxInflight = 2048;
            // return
            return ret;
        }

        private Exception internalSubscribe(string subject, string queue, string durable, EventHandler<StanMsgHandlerArgs> cb, out IStanSubscription sub)
        {
            sub = null;
            var fields = new Dictionary<string, object> {
                { "serverURL", _serverURL } ,
                {"clusterID", _clusterID },
                {"clientID", _clientID },
                {"subject", subject },
                {"queue", queue },
                {"durable", durable },
            };
            // check connection first
            var error = reconnect();
            if (error != null)
            {
                logWarn(fields, "nats subscribe failed at reconnect");
                return error;
            }
            // now subscribe
            try
            {
                var opts = getSubscriptionOptions(durable);
                if (string.IsNullOrEmpty(queue))
                    sub = _conn.Subscribe(subject, opts, cb);
                else
                    sub = _conn.Subscribe(subject, queue, opts, cb);
            }
            catch (Exception ex)
            {
                error = ex;
            }
            if (error != null)
            {
                fields["error"] = error;
                logWarn(fields, "nats subscribe failed");
            }
            else
            {
                logInfo(fields, "nats subscribe completed");
            }
            // return 
            return error;
        }

        public string Subscribe(string subject, string durable, EventHandler<StanMsgHandlerArgs> cb)
        {
            return QueueSubscribe(subject, "", durable, cb);
        }
        public string QueueSubscribe(string subject, string queue, string durable, EventHandler<StanMsgHandlerArgs> cb)
        {
            IStanSubscription sub = null;
            string guid = Guid.NewGuid().ToString();
            var error = internalSubscribe(subject, queue, durable, cb, out sub);
            // keep a copy of subscription info
            _subs[guid] = new SubRecord
            {
                subject = subject,
                queue = queue,
                durable = durable,
                cb = cb,
                sub = sub,
            };
            if (error != null)
            {
                internalClose();
                reconnect();
            }
            // return
            return guid;
        }
        private Exception removeSubscription(string guid, out SubRecord subRec)
        {
            lock (_token)
            {
                if (_subs.TryGetValue(guid, out subRec))
                {
                    _subs.Remove(guid);
                    return null;
                }
            }
            return new ApplicationException(string.Format("nats subscription not found. guid='{0}'", guid));
        }
        public void Unscribe(string guid)
        {
            SubRecord subRec;
            var error = removeSubscription(guid, out subRec);
            if (error != null)
                throw error;
            // close the subscription
            try
            {
                subRec.sub.Unsubscribe();
            }
            catch (Exception ex)
            {
                error = ex;
            }
            var fields = new Dictionary<string, object> {
                    { "guid", guid } ,
                };
            if (error != null)
            {
                fields["error"] = error;
                logWarn(fields, "nats unsubscribe failed");
            }
            else
            {
                logInfo(fields, "nats unsubscribe completed");
            }
        }
        public void Closesubscribe(string guid)
        {
            var fields = new Dictionary<string, object> {
                { "guid", guid } ,
            };
            SubRecord subRec;
            var error = removeSubscription(guid, out subRec);
            if (error != null)
                throw error;
            // close the subscription
            try
            {
                subRec.sub.Close();
            }
            catch (Exception ex)
            {
                error = ex;
            }
            if (error != null)
            {
                fields["error"] = error;
                logWarn(fields, "nats closesubscribe failed");
            }
            else
            {
                logInfo(fields, "nats closesubscribe completed");
            }
        }
        private string getLogText(string level, Dictionary<string, object> fields, string format, params object[] args)
        {
            var sb = new StringBuilder();
            var map = new Dictionary<string, object>();
            sb.AppendFormat("time=\"{0}\" level=\"{1}\" ", DateTime.Now, level);
            // add fields
            if (fields != null)
            {
                foreach (var field in fields)
                {
                    var text = field.Value == null ? "" : field.Value.ToString();
                    sb.AppendFormat("{0}=\"{1}\" ", field.Key, text.Replace("\"", "\\\""));
                }
            }
            var message = string.Format(format, args);
            sb.AppendFormat("message=\"{0}\"", message.Replace("\"", "\\\""));
            // return 
            return sb.ToString();
        }

        private void logInfo(Dictionary<string, object> fields, string format, params object[] args)
        {
            Logger.LogInformation(getLogText("info", fields, format, args));
        }
        private void logWarn(Dictionary<string, object> fields, string format, params object[] args)
        {
            Logger.LogWarning(getLogText("warn", fields, format, args));
        }

        private void logError(Dictionary<string, object> fields, string format, params object[] args)
        {
            Logger.LogError(getLogText("error", fields, format, args));
        }

        private void reconnectServer(object data)
        {
            var retries = 0;
            while (true)
            {
                if (_reconnectAbort.WaitOne(ReconnectDelay))
                    break;
                // check there is subscriptio or not
                if (_subs.Count > 0)
                {
                    if (_conn != null)
                    {
                        // ping the server
                        try
                        {
                            _conn.Publish("ping", null);
                        }
                        catch (Exception ex)
                        {
                            var fields = new Dictionary<string, object> { { "error", ex } };
                            logWarn(fields, "nats ping server failed, reconnect now...");
                            // close connection
                            internalClose();
                        }
                    }
                    // reconnect now
                    var error = reconnect();
                    if (error != null)
                    {
                        retries++;
                        var fields = new Dictionary<string, object> { { "retries", retries }, { "error", error } };
                        logWarn(fields, "nats reconnection failed, retry at {0}...", DateTime.Now + ReconnectDelay);
                    }
                }
            }
        }
    }

    public static class nats
    {
        public static int DefaultMaxInflight = 1;
        /// <summary>
        /// default subscribe ack wait time
        /// </summary>
        public static TimeSpan DefaultSubscribeAckWait = TimeSpan.FromSeconds(60);
        /// <summary>
        /// default reconnect delay time
        /// </summary>
        public static TimeSpan DefaultReconnectDelay = TimeSpan.FromSeconds(15);
        /// <summary>
        /// default publish retry delay time
        /// </summary>
        public static TimeSpan[] DefaultPublishRetryDelays = new TimeSpan[] { TimeSpan.FromSeconds(0), TimeSpan.FromSeconds(10), TimeSpan.FromSeconds(20) };
        /// <summary>
        /// default nats connection factory
        /// </summary>
        public static INatsFactory DefaultFactory = new NatsFactory();
        /// <summary>
        /// Default logger
        /// </summary>
        public static ILogger DefaultLogger = new LoggerFactory().AddConsole().CreateLogger<Nats>();
        /// <summary>
        /// default nats object
        /// </summary>
        private static object _defaultToken = new object();
        private static INats _default = null;
        public static INats Default
        {
            get
            {
                if (_default == null)
                {
                    lock (_defaultToken)
                    {
                        if (_default == null)
                            _default = new Nats(DefaultLogger)
                            {
                                MaxInflight = DefaultMaxInflight,
                                PublishRetryDelays = DefaultPublishRetryDelays,
                                ReconnectDelay = DefaultReconnectDelay,
                                SubscribeAckWait = DefaultSubscribeAckWait,
                            };
                    }
                }
                return _default;
            }
            set
            {
                lock (_defaultToken)
                    _default = value;
            }
        }

        /// <summary>
        /// Connect to nats server
        /// </summary>
        /// <param name="serverURL"></param>
        /// <param name="clusterID"></param>
        /// <param name="serviceID"></param>
        public static void Connect(string serverURL, string clusterID, string serviceID)
        {
            Default.Connect(serverURL, clusterID, serviceID);
        }
        /// <summary>
        /// close the connection
        /// </summary>
        public static void Close()
        {
            Default.Close();
            Default = null;
        }
        /// <summary>
        /// publish message to server with bytes
        /// </summary>
        /// <param name="subject"></param>
        /// <param name="data"></param>
        /// <param name="delays"></param>
        public static void Publish(string subject, byte[] data, TimeSpan[] delays = null)
        {
            Default.Publish(subject, data, delays);
        }
        /// <summary>
        /// publish message to server with string
        /// </summary>
        /// <param name="subject"></param>
        /// <param name="msg"></param>
        /// <param name="delays"></param>
        public static void Publish(string subject, string msg, TimeSpan[] delays = null)
        {
            Default.Publish(subject, msg, delays);
        }
        /// <summary>
        /// subscribe to a channel
        /// </summary>
        /// <param name="subject"></param>
        /// <param name="durable"></param>
        /// <param name="cb"></param>
        /// <returns></returns>
        public static string Subscribe(string subject, string durable, EventHandler<StanMsgHandlerArgs> cb)
        {
            return Default.Subscribe(subject, durable, cb);
        }
        /// <summary>
        /// queue subscribe to a channel
        /// </summary>
        /// <param name="subject"></param>
        /// <param name="queue"></param>
        /// <param name="durable"></param>
        /// <param name="cb"></param>
        /// <returns></returns>
        public static string QueueSubscribe(string subject, string queue, string durable, EventHandler<StanMsgHandlerArgs> cb)
        {
            return Default.QueueSubscribe(subject, queue, durable, cb);
        }
        /// <summary>
        /// unsubscribe to a channel
        /// </summary>
        /// <param name="guid"></param>
        public static void Unscribe(string guid)
        {
            Default.Unscribe(guid);
        }
        /// <summary>
        /// close a subscription
        /// </summary>
        /// <param name="guid"></param>
        public static void Closesubscribe(string guid)
        {
            Default.Closesubscribe(guid);
        }
    }

}
