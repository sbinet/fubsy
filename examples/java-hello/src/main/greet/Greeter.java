package greet;

class Greeter implements IGreeter {
    String makeGreeting(String who) {
        return "hello, " + who;
    }

    public void greet(String who) {
        System.out.println(makeGreeting(who));
    }
}

