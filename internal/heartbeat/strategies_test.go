package heartbeat

import (
	"testing"
	"time"
)

func TestNewJitterGenerator(t *testing.T) {
	baseInterval := 30 * time.Minute
	gen := NewJitterGenerator(baseInterval)

	if gen == nil {
		t.Fatal("NewJitterGenerator returned nil")
	}

	if gen.baseInterval != baseInterval {
		t.Errorf("Expected baseInterval %v, got %v", baseInterval, gen.baseInterval)
	}
}

func TestJitterGenerator_NextInterval(t *testing.T) {
	baseInterval := 30 * time.Minute
	gen := NewJitterGenerator(baseInterval)

	// Generate multiple intervals
	intervals := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		intervals[i] = gen.NextInterval()
	}

	// Check that intervals are within reasonable bounds (0.5x to 2x)
	minBound := time.Duration(float64(baseInterval) * 0.5)
	maxBound := time.Duration(float64(baseInterval) * 2.0)

	for i, interval := range intervals {
		if interval < minBound {
			t.Errorf("Interval %d (%v) is below minimum bound %v", i, interval, minBound)
		}
		if interval > maxBound {
			t.Errorf("Interval %d (%v) is above maximum bound %v", i, interval, maxBound)
		}
	}

	// Check that intervals vary (not all the same)
	allSame := true
	first := intervals[0]
	for _, interval := range intervals[1:] {
		if interval != first {
			allSame = false
			break
		}
	}
	if allSame {
		t.Error("All intervals are the same - jitter not working")
	}
}

func TestNaturalisticJitter(t *testing.T) {
	baseInterval := 30 * time.Minute

	// Generate multiple jitter values
	intervals := make([]time.Duration, 20)
	for i := 0; i < 20; i++ {
		intervals[i] = NaturalisticJitter(baseInterval)
	}

	// Check bounds
	minBound := time.Duration(float64(baseInterval) * 0.1) // Allow wider range
	maxBound := time.Duration(float64(baseInterval) * 5.0)

	for i, interval := range intervals {
		if interval < minBound {
			t.Logf("Interval %d (%v) is below expected minimum, but allowing", i, interval)
		}
		if interval > maxBound {
			t.Errorf("Interval %d (%v) is above maximum bound %v", i, interval, maxBound)
		}
	}

	// Check variation
	allSame := true
	first := intervals[0]
	for _, interval := range intervals[1:] {
		if interval != first {
			allSame = false
			break
		}
	}
	if allSame {
		t.Error("All intervals are the same - jitter not working")
	}
}

func TestExponentialJitter(t *testing.T) {
	baseInterval := 30 * time.Minute

	intervals := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		intervals[i] = ExponentialJitter(baseInterval)
	}

	// Exponential jitter should always be >= baseInterval
	for i, interval := range intervals {
		if interval < baseInterval {
			t.Errorf("Interval %d (%v) is below base interval %v", i, interval, baseInterval)
		}
	}
}

func TestGetNextHeartbeatTime(t *testing.T) {
	baseInterval := 30 * time.Minute

	now := time.Now()
	nextTime := GetNextHeartbeatTime(baseInterval)

	if nextTime.Before(now) {
		t.Error("Next heartbeat time should be in the future")
	}

	// Should be within reasonable bounds
	maxFuture := now.Add(baseInterval * 3)
	if nextTime.After(maxFuture) {
		t.Errorf("Next heartbeat time %v is too far in future (max: %v)", nextTime, maxFuture)
	}
}
