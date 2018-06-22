package pub;

import com.deluxe.one.nats.NatsStreamSubject;

public class SinglePublisher {
	public static void main(String[] args) {

		// For the publisher case, we do not care durable name
		NatsStreamSubject subject = new NatsStreamSubject("test-cluster",
				"nats://localhost:4222", "test-client", null, "test");

		// Synchronous call. Throws an exception if publish fails.
		subject.publish("1", new int[]{0, 5, 10}, false);
	}
}
