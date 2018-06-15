import com.deluxe.one.nats.NatsStreamSubject;

import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;

public class PublisherTest {
	public static void main(String[] args) {

		// For the publisher case, we do not care durable name
		NatsStreamSubject subject = new NatsStreamSubject("test-cluster",
				"nats://localhost:4222", null, "test");

		AtomicInteger counter = new AtomicInteger();
		Executors.newScheduledThreadPool(1).scheduleAtFixedRate(() -> {
			try {
				int current = counter.addAndGet(1);
				String data = "counter: " + current;
				subject.publish(data);
			} catch (Exception ex) {
				ex.printStackTrace();
			}
		}, 0,1, TimeUnit.SECONDS);
	}
}
