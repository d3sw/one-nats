package com.deluxe.one.nats;

import io.nats.streaming.StreamingConnection;
import io.nats.streaming.StreamingConnectionFactory;
import io.nats.streaming.Subscription;
import io.nats.streaming.SubscriptionOptions;
import org.apache.commons.lang3.StringUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.UUID;

/**
 * @author Oleksiy Lysak
 */
public class NatsStreamSubject extends NatsAbstractSubject {
	private static Logger logger = LoggerFactory.getLogger(NatsStreamSubject.class);
	private StreamingConnectionFactory fact;
	private StreamingConnection conn;
	private Subscription subs;
	private String durableName;

	public NatsStreamSubject(String clusterId, String natsUrl, String serviceName, String durableName, String name) {
		super(name);
		this.fact = new StreamingConnectionFactory();
		this.fact.setClusterId(clusterId);
		this.fact.setClientId(serviceName + "-" + UUID.randomUUID().toString());
		this.fact.setNatsUrl(natsUrl);
		this.durableName = durableName;
		open();
	}

	@Override
	public boolean isConnected() {
		return conn != null &&
				conn.getNatsConnection() != null &&
				conn.getNatsConnection().isConnected();
	}

	@Override
	protected void connect() {
		try {
			StreamingConnection temp = fact.createConnection();
			logger.info("Successfully connected for " + name);

			temp.getNatsConnection().setReconnectedCallback((event) ->
					logger.warn("onReconnect. Reconnected back for " + name));
			temp.getNatsConnection().setDisconnectedCallback((event ->
					logger.warn("onDisconnect. Disconnected for " + name)));

			conn = temp;
		} catch (Exception e) {
			logger.error("Unable to establish nats streaming connection for " + name, e);
			throw new RuntimeException(e);
		}
	}

	@Override
	protected void subscribe() {
		// do nothing if already subscribed
		if (subs != null) {
			return;
		}

		try {
			ensureConnected();

			SubscriptionOptions subscriptionOptions = new SubscriptionOptions
					.Builder().durableName(durableName).build();

			// Create subject/queue subscription if the queue has been provided
			if (StringUtils.isNotEmpty(queue)) {
				logger.info(String.format("No subscription. Creating a queue subscription. subject=%s, queue=%s", subject, queue));
				subs = conn.subscribe(subject, queue,
						msg -> onMessage(msg.getSubject(), msg.getData()),
						subscriptionOptions);
			} else {
				logger.info(String.format("No subscription. Creating a pub/sub subscription. subject=%s", subject));
				subs = conn.subscribe(subject,
						msg -> onMessage(msg.getSubject(), msg.getData()),
						subscriptionOptions);
			}
		} catch (Exception ex) {
			logger.error("Subscription failed with " + ex.getMessage() + " for name " + name, ex);
		}
	}

	@Override
	protected void publish(String subject, byte[] data) throws Exception {
		ensureConnected();
		conn.publish(subject, data);
	}

	@Override
	protected void closeSubs() {
		if (subs == null) {
			return;
		}
		try {
			subs.close(true);
		} catch (Exception ignore) {
		}
		subs = null;
	}

	@Override
	protected void closeConn() {
		if (conn == null) {
			return;
		}
		try {
			conn.close();
		} catch (Exception ignore) {
		}
		conn = null;
	}
}
