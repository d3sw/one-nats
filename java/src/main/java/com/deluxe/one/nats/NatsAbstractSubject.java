package com.deluxe.one.nats;

import io.reactivex.Observable;
import io.reactivex.ObservableOnSubscribe;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.LinkedList;
import java.util.List;
import java.util.concurrent.Executors;
import java.util.concurrent.LinkedBlockingQueue;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.locks.Lock;
import java.util.concurrent.locks.ReentrantLock;

/**
 * @author Oleksiy Lysak
 */
public abstract class NatsAbstractSubject {
	private static Logger logger = LoggerFactory.getLogger(NatsAbstractSubject.class);
	private LinkedBlockingQueue<String> messages = new LinkedBlockingQueue<>();
	private final Lock mu = new ReentrantLock();
	private ScheduledExecutorService execs;
	String subject;
	String queue;
	String name;

	// Indicates that observe was called (Event Handler) and we must to re-initiate subscription upon reconnection
	private boolean observable;
	private boolean isOpened;

	NatsAbstractSubject(String name) {
		this.name = name;

		// If queue specified (e.g. subject:queue) - split to subject & queue
		if (name.contains(":")) {
			this.subject = name.substring(0, name.indexOf(':'));
			this.queue = name.substring(name.indexOf(':') + 1);
		} else {
			this.subject = name;
			this.queue = null;
		}
		logger.info(String.format("Initialized with name=%s, subject=%s, queue=%s", name, subject, queue));
	}

	void onMessage(String subject, byte[] data) {
		String payload = new String(data);
		logger.info(String.format("Received message for %s: %s", subject, payload));

		messages.add(payload);
	}

	public Observable<String> listen() {
		logger.info("Listen started for " + name);
		observable = true;

		mu.lock();
		try {
			subscribe();
		} finally {
			mu.unlock();
		}

		ObservableOnSubscribe<String> onSubscribe = subscriber -> {
			Observable<Long> interval = Observable.interval(100, TimeUnit.MILLISECONDS);
			interval.flatMap((Long x) -> {
				List<String> available = new LinkedList<>();
				messages.drainTo(available);
				return Observable.fromIterable(available);
			}).subscribe(subscriber::onNext, subscriber::onError);
		};

		return Observable.create(onSubscribe);
	}

	public void publish(String payload) {
		try {
			publish(subject, payload.getBytes());
			logger.info(String.format("Published message to %s: %s", subject, payload));
		} catch (Exception ex) {
			logger.error("Failed to publish message " + payload + " to " + subject, ex);
			throw new RuntimeException(ex);
		}
	}

	public void close() {
		logger.info("Closing connection for " + name);
		mu.lock();
		try {
			if (execs != null) {
				execs.shutdownNow();
				execs = null;
			}
			closeSubs();
			closeConn();
			isOpened = false;
		} finally {
			mu.unlock();
		}
	}

	public void open() {
		// do nothing if not closed
		if (isOpened) {
			return;
		}

		mu.lock();
		try {
			try {
				connect();

				// Re-initiated subscription if existed
				if (observable) {
					subscribe();
				}
			} catch (Exception ignore) {
			}

			execs = Executors.newScheduledThreadPool(1);
			execs.scheduleAtFixedRate(this::guard, 0, 500, TimeUnit.MILLISECONDS);
			isOpened = true;
		} finally {
			mu.unlock();
		}
	}

	private void guard() {
		if (isConnected()) {
			return;
		}

		logger.error("Guard invoked for " + name);
		mu.lock();
		try {
			closeSubs();
			closeConn();

			// Connect
			connect();

			// Re-initiated subscription if existed
			if (observable) {
				subscribe();
			}
		} catch (Exception ex) {
			logger.error("Guard failed with " + ex.getMessage() + " for " + name, ex);
		} finally {
			mu.unlock();
		}
	}

	void ensureConnected() {
		if (!isConnected()) {
			throw new RuntimeException("No nats connection");
		}
	}

	abstract void connect();

	abstract boolean isConnected();

	protected void publish(String subject, byte[] data) throws Exception {
	}

	protected void subscribe() {
	}

	protected void closeSubs() {
	}

	protected void closeConn() {
	}
}
