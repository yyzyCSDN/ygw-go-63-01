package metric

import "testing"

func TestCountersAndGauges(t *testing.T) {
	metrics := New()
	counter := metrics.Counter("a.b")
	counter.Inc()
	counter.Add(4)
	if counter.Value() != 5 {
		t.Fatalf("counter value = %d, want 5", counter.Value())
	}
	gauge := metrics.Gauge("c.d")
	gauge.Set(7)
	gauge.Add(3)
	if gauge.Value() != 10 {
		t.Fatalf("gauge value = %d, want 10", gauge.Value())
	}
	snapshot := metrics.Snapshot()
	if snapshot["a.b"] != 5 || snapshot["c.d"] != 10 {
		t.Fatalf("unexpected snapshot: %v", snapshot)
	}
	names := metrics.Names()
	if len(names) != 2 || names[0] != "a.b" || names[1] != "c.d" {
		t.Fatalf("unexpected names: %v", names)
	}
}

func TestRecorder(t *testing.T) {
	metrics := New()
	rec := NewRecorder(metrics)
	rec.Inc(NamePublish)
	rec.Add(NameRoute, 3)
	rec.Set(NameConnInFlight, 2)
	if metrics.Counter(NamePublish).Value() != 1 {
		t.Fatal("recorder Inc did not reach metrics")
	}
	if metrics.Counter(NameRoute).Value() != 3 {
		t.Fatal("recorder Add did not reach metrics")
	}
	if metrics.Gauge(NameConnInFlight).Value() != 2 {
		t.Fatal("recorder Set did not reach metrics")
	}
	null := NullRecorder{}
	null.Inc("x")
	null.Add("x", 1)
	null.Set("x", 1)
}
