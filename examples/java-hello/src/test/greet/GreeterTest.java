package greet;

import org.junit.Test;
import static org.junit.Assert.*;

public class GreeterTest {
    @Test public void test_makeGreeting() {
        assertEquals("hello, alien overlords",
                     new Greeter().makeGreeting("alien overlords"));
    }
}
