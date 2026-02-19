package heartbeat

import (
	"math"
	"math/rand"
	"time"
)

// JitterGenerator generates naturalistic jitter for heartbeats
type JitterGenerator struct {
	baseInterval time.Duration
	lastInterval time.Duration
}

// NewJitterGenerator creates a new jitter generator
func NewJitterGenerator(baseInterval time.Duration) *JitterGenerator {
	return &JitterGenerator{
		baseInterval: baseInterval,
		lastInterval: baseInterval,
	}
}

// NextInterval returns the next interval with naturalistic jitter
func (j *JitterGenerator) NextInterval() time.Duration {
	// Use log-normal distribution for more natural timing
	// This creates intervals that look more like normal network traffic
	mu := math.Log(float64(j.baseInterval))
	sigma := 0.3 // Controls spread

	// Generate log-normal random value
	normal := rand.NormFloat64()
	logNormal := math.Exp(mu + sigma*normal)

	// Clamp to reasonable bounds (0.5x to 2x base interval)
	minInterval := time.Duration(float64(j.baseInterval) * 0.5)
	maxInterval := time.Duration(float64(j.baseInterval) * 2.0)

	interval := time.Duration(logNormal)
	if interval < minInterval {
		interval = minInterval
	}
	if interval > maxInterval {
		interval = maxInterval
	}

	j.lastInterval = interval
	return interval
}

// ExponentialJitter uses exponential distribution
func ExponentialJitter(baseInterval time.Duration) time.Duration {
	lambda := 1.0 / float64(baseInterval)
	interval := time.Duration(-math.Log(rand.Float64()) / lambda)

	// Add random component (0-10 minutes)
	randomComponent := time.Duration(rand.Intn(600)) * time.Second
	return baseInterval + randomComponent + interval
}

// NaturalisticJitter creates jitter that blends with network noise
func NaturalisticJitter(baseInterval time.Duration) time.Duration {
	// Use multiple distributions for variety
	method := rand.Intn(3)

	switch method {
	case 0:
		// Log-normal (most common)
		mu := math.Log(float64(baseInterval))
		sigma := 0.25
		normal := rand.NormFloat64()
		return time.Duration(math.Exp(mu + sigma*normal))
	case 1:
		// Exponential with base
		return ExponentialJitter(baseInterval)
	default:
		// Uniform with slight bias
		jitter := time.Duration(rand.Intn(600)) * time.Second
		return baseInterval + jitter
	}
}

// GetNextHeartbeatTime calculates next heartbeat time with jitter
func GetNextHeartbeatTime(baseInterval time.Duration) time.Time {
	jitter := NaturalisticJitter(baseInterval)
	return time.Now().Add(jitter)
}
