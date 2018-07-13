package com.deluxe.one.nats;

import io.nats.client.Connection;
import io.nats.client.ConnectionFactory;
import io.nats.client.Subscription;
import org.apache.commons.lang3.StringUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * @author Oleksiy Lysak
 */
public class NatsSubject extends NatsAbstractSubject {
	private static Logger logger = LoggerFactory.getLogger(NatsSubject.class);
	private ConnectionFactory fact;
	private Subscription subs;
	private Connection conn;

	public NatsSubject(ConnectionFactory factory, String name) {
		super(name);

		this.fact = factory;
		open();
	}

	@Override
	public boolean isConnected() {
		return conn != null &&
				conn.isConnected();
	}

	@Override
	protected void connect() {
		try {
			Connection temp = fact.createConnection();
			logger.info("Successfully connected for " + name);

			temp.setReconnectedCallback((event) -> logger.warn("onReconnect. Reconnected back for " + name));
			temp.setDisconnectedCallback((event -> logger.warn("onDisconnect. Disconnected for " + name)));

			conn = temp;
		} catch (Exception e) {
			logger.error("Unable to establish nats connection for " + name, e);
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

			// Create subject/queue subscription if the queue has been provided
			if (StringUtils.isNotEmpty(queue)) {
				logger.info(String.format("No subscription. Creating a queue subscription. subject=%s, queue=%s", subject, queue));
				subs = conn.subscribe(subject, queue, msg -> onMessage(msg.getSubject(), msg.getData()));
			} else {
				logger.info(String.format("No subscription. Creating a pub/sub subscription. subject=%s", subject));
				subs = conn.subscribe(subject, msg -> onMessage(msg.getSubject(), msg.getData()));
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
			subs.close();
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
