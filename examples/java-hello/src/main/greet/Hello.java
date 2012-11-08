package greet;

public class Hello {
    public static void main(String[] args) {
        IGreeter greeter = new Greeter();
        greeter.greet("world");
    }
}
