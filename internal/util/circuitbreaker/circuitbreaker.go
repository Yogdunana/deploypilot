package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State int

const (
	Closed   State = iota // Normal operation, requests pass through
	Open                  // Failing, requests are rejected
	HalfOpen              // Testing if the service recovered
)

func (s State) String() string {
	switch s {
	case Closed:
		return "closed"
	case Open:
		return "open"
	case HalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

var (
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// Config holds circuit breaker configuration.
type Config struct {
	// FailureThreshold is the number of failures before opening the circuit.
	FailureThreshold int
	// SuccessThreshold is the number of successes in half-open before closing.
	SuccessThreshold int
	// Timeout is how long the circuit stays open before transitioning to half-open.
	Timeout time.Duration
	// OnStateChange is an optional callback when state changes.
	OnStateChange func(from, to State)
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	}
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	mu              sync.RWMutex
	config          Config
	state           State
	failures        int
	successes       int
	lastFailureTime time.Time
}

// New creates a new CircuitBreaker with the given config.
func New(config Config) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  Closed,
	}
}

// State returns the current state (thread-safe).
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	if cb.state == Open && time.Since(cb.lastFailureTime) > cb.config.Timeout {
		return HalfOpen
	}
	return cb.state
}

// Execute runs the given function through the circuit breaker.
// If the circuit is open, it returns ErrCircuitOpen immediately.
// On success, it records a success. On failure, it records a failure.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	state := cb.State()
	if state == Open {
		return ErrCircuitOpen
	}

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.successes = 0
		cb.lastFailureTime = time.Now()
		if cb.failures >= cb.config.FailureThreshold {
			cb.transition(Open)
		}
		return err
	}

	cb.successes++
	if state == HalfOpen && cb.successes >= cb.config.SuccessThreshold {
		cb.failures = 0
		cb.successes = 0
		cb.transition(Closed)
	}
	return nil
}

func (cb *CircuitBreaker) transition(to State) {
	from := cb.state
	cb.state = to
	if cb.config.OnStateChange != nil && from != to {
		cb.config.OnStateChange(from, to)
	}
}

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = Closed
	cb.failures = 0
	cb.successes = 0
}
