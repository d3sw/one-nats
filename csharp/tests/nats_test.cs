using Microsoft.VisualStudio.TestTools.UnitTesting;
using Moq;
using Deluxe.One.Nats;
using STAN.Client;
using Microsoft.Extensions.Logging;
using System;
using System.Text;

namespace tests
{
    public class TextLogger : ILogger
    {
        private StringBuilder _sb = new StringBuilder();
        public IDisposable BeginScope<TState>(TState state)
        {
            return null;
        }

        public bool IsEnabled(LogLevel logLevel)
        {
            return true;
        }

        public void Log<TState>(LogLevel logLevel, EventId eventId, TState state, Exception exception, Func<TState, Exception, string> formatter)
        {
            _sb.AppendFormat("{0}: {1} - {2}", logLevel.ToString(), eventId.Id, formatter(state, exception));
        }
        public override string ToString()
        {
            return _sb.ToString();
        }
    }

    [TestClass]
    public class nats_test
    {
        [TestMethod]
        public void Test_Connect_Error()
        {
            var errmsg = "nats connection failed";
            var logger = new TextLogger();
            // mock
            var mockFactory = new Mock<INatsFactory>();
            mockFactory.Setup(m => m.CreateConnection(It.IsAny<string>(), It.IsAny<string>(), It.IsAny<StanOptions>())).Returns((IStanConnection)null);
            nats.DefaultFactory = mockFactory.Object;
            nats.DefaultLogger = logger;
            // run
            nats.Connect("", "", "");
            nats.Close();
            // verify
            StringAssert.Contains(logger.ToString(), errmsg);
        }

        [TestMethod]
        public void Test_Connect_Exception()
        {
            var errmsg = "error at connection";
            var logger = new TextLogger();
            // mock
            var mockFactory = new Mock<INatsFactory>();
            mockFactory.Setup(m => m.CreateConnection(It.IsAny<string>(), It.IsAny<string>(), It.IsAny<StanOptions>()))
                .Callback(() =>
                {
                    throw new Exception(errmsg);
                });
            nats.DefaultFactory = mockFactory.Object;
            nats.DefaultLogger = logger;
            // run
            nats.Connect("", "", "");
            nats.Close();
            // verify
            StringAssert.Contains(logger.ToString(), errmsg);
        }

        [TestMethod]
        public void Test_Connect_OK()
        {
            var logger = new TextLogger();
            // mock
            var mockFactory = new Mock<INatsFactory>();
            var mockConn = new Mock<IStanConnection>();
            mockFactory.Setup(m => m.CreateConnection(It.IsAny<string>(), It.IsAny<string>(), It.IsAny<StanOptions>())).Returns(mockConn.Object);
            nats.DefaultFactory = mockFactory.Object;
            nats.DefaultLogger = logger;
            // run
            nats.Connect("", "", "");
            nats.Close();
            // verify
            StringAssert.Contains(logger.ToString(), "nats connection completed");
        }

        [TestMethod]
        public void Test_Publish_Subject_Null()
        {
            var logger = new TextLogger();
            Exception error = null;
            // mock
            var mockConn = new Mock<IStanConnection>();
            var mockFactory = new Mock<INatsFactory>();
            mockFactory.Setup(m => m.CreateConnection(It.IsAny<string>(), It.IsAny<string>(), It.IsAny<StanOptions>())).Returns(mockConn.Object);
            nats.DefaultFactory = mockFactory.Object;
            nats.DefaultLogger = logger;
            // run
            nats.Connect("", "", "");
            try
            {
                nats.Publish(null, "message");
            }
            catch (Exception ex){ error = ex; }
            nats.Close();
            // verify
            Assert.IsNotNull(error, "should have error");
            StringAssert.Contains(error.ToString(), "invalid parameter");
        }

        [TestMethod]
        public void Test_Publish_Subject_Empty()
        {
            var logger = new TextLogger();
            Exception error = null;
            // mock
            var mockConn = new Mock<IStanConnection>();
            var mockFactory = new Mock<INatsFactory>();
            mockFactory.Setup(m => m.CreateConnection(It.IsAny<string>(), It.IsAny<string>(), It.IsAny<StanOptions>())).Returns(mockConn.Object);
            nats.DefaultFactory = mockFactory.Object;
            nats.DefaultLogger = logger;
            // run
            nats.Connect("", "", "");
            try
            {
                nats.Publish("", "message");
            }
            catch (Exception ex) { error = ex; }
            nats.Close();
            // verify
            Assert.IsNotNull(error, "should have error");
            StringAssert.Contains(error.ToString(), "invalid parameter");
        }

        [TestMethod]
        public void Test_Publish_Error()
        {
            var errmsg = "nats publish error";
            var logger = new TextLogger();
            // mock
            var mockConn = new Mock<IStanConnection>();
            var mockFactory = new Mock<INatsFactory>();
            mockFactory.Setup(m => m.CreateConnection(It.IsAny<string>(), It.IsAny<string>(), It.IsAny<StanOptions>())).Returns(mockConn.Object);
            mockConn.Setup(m => m.Publish(It.IsAny<string>(), It.IsAny<byte[]>())).Callback(() =>
            {
                throw new Exception(errmsg);
            });
            nats.DefaultPublishRetryDelays = new TimeSpan[] { TimeSpan.FromMilliseconds(1) };
            nats.DefaultFactory = mockFactory.Object;
            nats.DefaultLogger = logger;
            // run
            nats.Connect("", "", "");
            try
            {
                nats.Publish("subject", "message");
            }
            catch { }
            nats.Close();
            // verify
            StringAssert.Contains(logger.ToString(), errmsg);
        }

        [TestMethod]
        public void Test_Publish_OK()
        {
            string msg = "this is sample message", msgSaved = "";
            var logger = new TextLogger();
            // mock
            var mockConn = new Mock<IStanConnection>();
            var mockFactory = new Mock<INatsFactory>();
            mockFactory.Setup(m => m.CreateConnection(It.IsAny<string>(), It.IsAny<string>(), It.IsAny<StanOptions>())).Returns(mockConn.Object);
            mockConn.Setup(m => m.Publish(It.IsAny<string>(), It.IsAny<byte[]>())).Callback<string, byte[]>((sub, data) =>
            {
                msgSaved = Encoding.UTF8.GetString(data);
            });
            nats.DefaultPublishRetryDelays = new TimeSpan[] { TimeSpan.FromMilliseconds(1) };
            nats.DefaultFactory = mockFactory.Object;
            nats.DefaultLogger = logger;
            // run
            nats.Connect("", "", "");
            try
            {
                nats.Publish("subject", msg);
            }
            catch { }
            nats.Close();
            // verify
            Assert.AreEqual(msg, msgSaved);
            StringAssert.Contains(logger.ToString(), "nats publish completed");
        }

        [TestMethod]
        public void Test_Subscribe_Error()
        {
            var errmsg = "nats subscribe error";
            var logger = new TextLogger();
            // mock
            var mockConn = new Mock<IStanConnection>();
            var mockFactory = new Mock<INatsFactory>();
            mockFactory.Setup(m => m.CreateConnection(It.IsAny<string>(), It.IsAny<string>(), It.IsAny<StanOptions>())).Returns(mockConn.Object);
            mockConn.Setup(m => m.Subscribe(It.IsAny<string>(), It.IsAny<StanSubscriptionOptions>(), It.IsAny<EventHandler<StanMsgHandlerArgs>>())).Callback(() =>
           {
               throw new Exception(errmsg);
           });
            nats.DefaultPublishRetryDelays = new TimeSpan[] { TimeSpan.FromMilliseconds(1) };
            nats.DefaultFactory = mockFactory.Object;
            nats.DefaultLogger = logger;
            // run
            nats.Connect("", "", "");
            nats.Subscribe("subject", "durable", (e, args) => { });
            nats.Close();
            // verify
            StringAssert.Contains(logger.ToString(), errmsg);
        }

        [TestMethod]
        public void Test_Subscribe_OK()
        {
            var logger = new TextLogger();
            // mock
            var mockSub = new Mock<IStanSubscription>();
            var mockConn = new Mock<IStanConnection>();
            var mockFactory = new Mock<INatsFactory>();
            mockFactory.Setup(m => m.CreateConnection(It.IsAny<string>(), It.IsAny<string>(), It.IsAny<StanOptions>())).Returns(mockConn.Object);
            mockConn.Setup(m => m.Subscribe(It.IsAny<string>(), It.IsAny<StanSubscriptionOptions>(), It.IsAny<EventHandler<StanMsgHandlerArgs>>())).Returns(mockSub.Object);
            nats.DefaultFactory = mockFactory.Object;
            nats.DefaultLogger = logger;
            // run
            nats.Connect("", "", "");
            nats.Subscribe("subject", "durable", (e, args) => { });
            nats.Close();
            // verify
            StringAssert.Contains(logger.ToString(), "nats subscribe completed");
        }

        [TestMethod]
        public void Test_QueueSubscribe_Error()
        {
            var errmsg = "nats subscribe error";
            var logger = new TextLogger();
            // mock
            var mockConn = new Mock<IStanConnection>();
            var mockFactory = new Mock<INatsFactory>();
            mockFactory.Setup(m => m.CreateConnection(It.IsAny<string>(), It.IsAny<string>(), It.IsAny<StanOptions>())).Returns(mockConn.Object);
            mockConn.Setup(m => m.Subscribe(It.IsAny<string>(), It.IsAny<string>(), It.IsAny<StanSubscriptionOptions>(), It.IsAny<EventHandler<StanMsgHandlerArgs>>())).Callback(() =>
            {
                throw new Exception(errmsg);
            });
            nats.DefaultPublishRetryDelays = new TimeSpan[] { TimeSpan.FromMilliseconds(1) };
            nats.DefaultFactory = mockFactory.Object;
            nats.DefaultLogger = logger;
            // run
            nats.Connect("", "", "");
            nats.QueueSubscribe("subject", "queue", "durable", (e, args) => { });
            nats.Close();
            // verify
            StringAssert.Contains(logger.ToString(), errmsg);
        }

        [TestMethod]
        public void Test_QueueSubscribe_OK()
        {
            var logger = new TextLogger();
            // mock
            var mockSub = new Mock<IStanSubscription>();
            var mockConn = new Mock<IStanConnection>();
            var mockFactory = new Mock<INatsFactory>();
            mockFactory.Setup(m => m.CreateConnection(It.IsAny<string>(), It.IsAny<string>(), It.IsAny<StanOptions>())).Returns(mockConn.Object);
            mockConn.Setup(m => m.Subscribe(It.IsAny<string>(), It.IsAny<string>(), It.IsAny<StanSubscriptionOptions>(), It.IsAny<EventHandler<StanMsgHandlerArgs>>())).Returns(mockSub.Object);
            nats.DefaultFactory = mockFactory.Object;
            nats.DefaultLogger = logger;
            // run
            nats.Connect("", "", "");
            nats.QueueSubscribe("subject", "queue", "durable", (e, args) => { });
            nats.Close();
            // verify
            StringAssert.Contains(logger.ToString(), "nats subscribe completed");
        }
    }
}
