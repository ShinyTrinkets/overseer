package overseer

import (
	"reflect"
	"testing"
	"time"
)

func TestBoff1(t *testing.T) {

	b := &Backoff{
		Min:    100 * time.Millisecond,
		Max:    10 * time.Second,
		Factor: 2,
	}

	equals(t, b.Duration(), 100*time.Millisecond)
	equals(t, b.Duration(), 200*time.Millisecond)
	equals(t, b.Duration(), 400*time.Millisecond)
	b.Reset()
	equals(t, b.Duration(), 100*time.Millisecond)
}

func TestBoff2(t *testing.T) {

	b := &Backoff{
		Min:    100 * time.Millisecond,
		Max:    10 * time.Second,
		Factor: 1.5,
	}

	equals(t, b.Duration(), 100*time.Millisecond)
	equals(t, b.Duration(), 150*time.Millisecond)
	equals(t, b.Duration(), 225*time.Millisecond)
	b.Reset()
	equals(t, b.Duration(), 100*time.Millisecond)
}

func TestBoff3(t *testing.T) {

	b := &Backoff{
		Min:    100 * time.Nanosecond,
		Max:    10 * time.Second,
		Factor: 1.75,
	}

	equals(t, b.Duration(), 100*time.Nanosecond)
	equals(t, b.Duration(), 175*time.Nanosecond)
	equals(t, b.Duration(), 306*time.Nanosecond)
	b.Reset()
	equals(t, b.Duration(), 100*time.Nanosecond)
}

func TestBoff4(t *testing.T) {
	b := &Backoff{
		Min:    500 * time.Second,
		Max:    100 * time.Second,
		Factor: 1,
	}

	equals(t, b.Duration(), b.Max)
}

func TestBoff5(t *testing.T) {

	b := &Backoff{
		Min:    10 * time.Nanosecond,
		Max:    30 * time.Second,
		Factor: 0.5,
	}

	equals(t, b.Duration(), 10*time.Nanosecond)
	equals(t, b.Duration(), 10*time.Nanosecond)
	equals(t, b.Duration(), 10*time.Nanosecond)
	equals(t, b.Duration(), 10*time.Nanosecond)
	equals(t, b.Duration(), 10*time.Nanosecond)
	b.Reset()
	equals(t, b.Duration(), 10*time.Nanosecond)
}

func TestBoffForAttempt(t *testing.T) {
	b := &Backoff{
		Min:    100 * time.Millisecond,
		Max:    10 * time.Second,
		Factor: 2,
	}

	equals(t, b.ForAttempt(0), 100*time.Millisecond)
	equals(t, b.ForAttempt(1), 200*time.Millisecond)
	equals(t, b.ForAttempt(2), 400*time.Millisecond)
	b.Reset()
	equals(t, b.ForAttempt(0), 100*time.Millisecond)
}

func TestBoffForAttemptDefaults(t *testing.T) {
	b := &Backoff{}

	equals(t, b.ForAttempt(0), 100*time.Millisecond)
	equals(t, b.ForAttempt(1), 200*time.Millisecond)
	equals(t, b.ForAttempt(2), 400*time.Millisecond)
	b.Reset()
	equals(t, b.ForAttempt(0), 100*time.Millisecond)
}

func TestBoffGetAttempt(t *testing.T) {
	b := &Backoff{
		Min:    100 * time.Millisecond,
		Max:    10 * time.Second,
		Factor: 2,
	}
	equals(t, b.Attempt(), float64(0))
	equals(t, b.Duration(), 100*time.Millisecond)
	equals(t, b.Attempt(), float64(1))
	equals(t, b.Duration(), 200*time.Millisecond)
	equals(t, b.Attempt(), float64(2))
	equals(t, b.Duration(), 400*time.Millisecond)
	equals(t, b.Attempt(), float64(3))
	b.Reset()
	equals(t, b.Attempt(), float64(0))
	equals(t, b.Duration(), 100*time.Millisecond)
	equals(t, b.Attempt(), float64(1))
}

func TestBoffJitter(t *testing.T) {
	b := &Backoff{
		Min:    100 * time.Millisecond,
		Max:    10 * time.Second,
		Factor: 2,
		Jitter: true,
	}

	equals(t, b.Duration(), 100*time.Millisecond)
	between(t, b.Duration(), 100*time.Millisecond, 200*time.Millisecond)
	between(t, b.Duration(), 100*time.Millisecond, 400*time.Millisecond)
	b.Reset()
	equals(t, b.Duration(), 100*time.Millisecond)
}

func between(t *testing.T, actual, low, high time.Duration) {
	if actual < low {
		t.Fatalf("Got %s, Expecting >= %s", actual, low)
	}
	if actual > high {
		t.Fatalf("Got %s, Expecting <= %s", actual, high)
	}
}

func equals(t *testing.T, v1, v2 interface{}) {
	if !reflect.DeepEqual(v1, v2) {
		t.Fatalf("Got %v, Expecting %v", v1, v2)
	}
}
