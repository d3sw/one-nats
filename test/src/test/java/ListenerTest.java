import com.deluxe.one.nats.NatsStreamSubject;
import io.reactivex.disposables.Disposable;

public class ListenerTest {
	public static void main(String[] args) throws Exception {
		NatsStreamSubject subject = new NatsStreamSubject("test-cluster",
				"nats://localhost:4222", "durable", "test:queue");

		// Start listening and subscribe for the data
		Disposable disposable = subject.listen().subscribe(m -> {
			System.out.println("Got payload = " + m + "\n");
		});

		// Want to un-subscribe? Just call the following:
		// disposable.dispose();
	}
}
