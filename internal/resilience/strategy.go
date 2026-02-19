package resilience

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Strategy represents a connection strategy
type Strategy struct {
	Name        string
	URL         string
	Priority    int
	LastSuccess time.Time
	LastFailure time.Time
	SuccessRate float64
	Attempts    int
	Successes   int
}

// StrategyManager manages multiple connection strategies
type StrategyManager struct {
	strategies []*Strategy
	current    int
	learned    map[string]*Strategy
}

// NewStrategyManager creates a new strategy manager
func NewStrategyManager(baseURL string) *StrategyManager {
	sm := &StrategyManager{
		learned: make(map[string]*Strategy),
	}

	// Initialize default strategies
	sm.strategies = []*Strategy{
		{Name: "HTTPS-443", URL: baseURL, Priority: 1, SuccessRate: 1.0},
		{Name: "HTTPS-8443", URL: replacePort(baseURL, 8443), Priority: 2, SuccessRate: 0.8},
		{Name: "HTTP-80", URL: replaceScheme(baseURL, "http"), Priority: 3, SuccessRate: 0.7},
		{Name: "HTTP-8080", URL: replaceSchemePort(baseURL, "http", 8080), Priority: 4, SuccessRate: 0.6},
		{Name: "HTTPS-4443", URL: replacePort(baseURL, 4443), Priority: 5, SuccessRate: 0.5},
		{Name: "HTTPS-8888", URL: replacePort(baseURL, 8888), Priority: 6, SuccessRate: 0.4},
		{Name: "HTTPS-9000", URL: replacePort(baseURL, 9000), Priority: 7, SuccessRate: 0.3},
		{Name: "HTTPS-9443", URL: replacePort(baseURL, 9443), Priority: 8, SuccessRate: 0.2},
	}

	return sm
}

// GetNextStrategy returns the next strategy to try
func (sm *StrategyManager) GetNextStrategy() *Strategy {
	if len(sm.strategies) == 0 {
		return nil
	}

	// Sort by priority and success rate
	best := sm.strategies[0]
	for _, s := range sm.strategies {
		score := s.SuccessRate * float64(s.Priority)
		bestScore := best.SuccessRate * float64(best.Priority)
		if score > bestScore {
			best = s
		}
	}

	return best
}

// RecordSuccess records a successful connection
func (sm *StrategyManager) RecordSuccess(strategyName string) {
	for _, s := range sm.strategies {
		if s.Name == strategyName {
			s.LastSuccess = time.Now()
			s.Attempts++
			s.Successes++
			s.SuccessRate = float64(s.Successes) / float64(s.Attempts)
			return
		}
	}
}

// RecordFailure records a failed connection
func (sm *StrategyManager) RecordFailure(strategyName string) {
	for _, s := range sm.strategies {
		if s.Name == strategyName {
			s.LastFailure = time.Now()
			s.Attempts++
			s.SuccessRate = float64(s.Successes) / float64(s.Attempts)
			return
		}
	}
}

// TryWithStrategies attempts a request with multiple strategies
func (sm *StrategyManager) TryWithStrategies(reqFunc func(url string) (*http.Response, error)) (*http.Response, error) {
	var lastErr error

	for i := 0; i < len(sm.strategies); i++ {
		strategy := sm.GetNextStrategy()
		if strategy == nil {
			break
		}

		resp, err := reqFunc(strategy.URL)
		if err == nil && resp != nil && resp.StatusCode < 500 {
			sm.RecordSuccess(strategy.Name)
			return resp, nil
		}

		lastErr = err
		sm.RecordFailure(strategy.Name)

		// Exponential backoff
		time.Sleep(time.Duration(i+1) * time.Second)
	}

	return nil, fmt.Errorf("all strategies failed: %w", lastErr)
}

// Helper functions
func replacePort(urlStr string, port int) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}
	u.Host = fmt.Sprintf("%s:%d", u.Hostname(), port)
	return u.String()
}

func replaceScheme(urlStr string, scheme string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}
	u.Scheme = scheme
	return u.String()
}

func replaceSchemePort(urlStr string, scheme string, port int) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}
	u.Scheme = scheme
	u.Host = fmt.Sprintf("%s:%d", u.Hostname(), port)
	return u.String()
}

