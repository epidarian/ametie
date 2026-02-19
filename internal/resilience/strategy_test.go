package resilience

import (
	"fmt"
	"net/http"
	"testing"
)

func TestNewStrategyManager(t *testing.T) {
	sm := NewStrategyManager("https://example.com")
	if sm == nil {
		t.Fatal("NewStrategyManager returned nil")
	}
	if len(sm.strategies) == 0 {
		t.Error("StrategyManager should have default strategies")
	}
}

func TestGetNextStrategy(t *testing.T) {
	sm := NewStrategyManager("https://example.com")

	strategy := sm.GetNextStrategy()
	if strategy == nil {
		t.Fatal("GetNextStrategy returned nil")
	}
	if strategy.Name == "" {
		t.Error("Strategy should have a name")
	}
	if strategy.URL == "" {
		t.Error("Strategy should have a URL")
	}
}

func TestRecordSuccess(t *testing.T) {
	sm := NewStrategyManager("https://example.com")
	strategy := sm.GetNextStrategy()
	originalSuccesses := strategy.Successes

	sm.RecordSuccess(strategy.Name)

	if strategy.Successes != originalSuccesses+1 {
		t.Errorf("Expected successes %d, got %d", originalSuccesses+1, strategy.Successes)
	}
	if strategy.Attempts != originalSuccesses+1 {
		t.Errorf("Expected attempts %d, got %d", originalSuccesses+1, strategy.Attempts)
	}
}

func TestRecordFailure(t *testing.T) {
	sm := NewStrategyManager("https://example.com")
	strategy := sm.GetNextStrategy()
	originalAttempts := strategy.Attempts
	originalSuccesses := strategy.Successes

	sm.RecordFailure(strategy.Name)

	if strategy.Attempts != originalAttempts+1 {
		t.Errorf("Expected attempts %d, got %d", originalAttempts+1, strategy.Attempts)
	}
	if strategy.Successes != originalSuccesses {
		t.Errorf("Successes should remain %d, got %d", originalSuccesses, strategy.Successes)
	}
}

func TestTryWithStrategies(t *testing.T) {
	sm := NewStrategyManager("https://example.com")

	// Test with a function that always fails
	reqFunc := func(url string) (*http.Response, error) {
		return nil, fmt.Errorf("connection failed")
	}

	resp, err := sm.TryWithStrategies(reqFunc)
	if err == nil {
		t.Error("Expected error when all strategies fail")
	}
	if resp != nil {
		t.Error("Response should be nil on failure")
	}
}
