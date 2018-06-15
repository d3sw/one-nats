package sub;

import com.deluxe.one.nats.NatsStreamSubject;
import io.reactivex.disposables.Disposable;

public class SimpleListener {
	public static void main(String[] args) throws Exception {
		NatsStreamSubject subject = new NatsStreamSubject("test-cluster",
				"nats://localhost:4222", "durable", "test:queue");

		// Name format is "subject name: queue group name".
		// If no queue group - the pub/sub mode used, otherwise the queue mode
		/*
		Examples:
		  "test" - just use pub/sub mode for the `test` subject.
		  "test:queue" - use queue mode for `test` subject and queue group name `queue`
		*/

		// Start listening and subscribe for the data
		Disposable subscribe = subject.listen().subscribe(m -> {
			System.out.println("Got payload = " + m + "\n");
		});
	}
}
