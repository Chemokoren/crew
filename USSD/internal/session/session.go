// Package session provides ultra-fast, Redis-backed USSD session management.
// Sessions are the critical state bridge in USSD interactions — each telco request
// arrives independently, and the session store reconstructs user context.
//
// Design decisions:
//   - Redis for sub-millisecond lookup at scale
//   - Minimal session payload (state + last input + metadata) for low memory
//   - TTL-based expiry aligned with telco timeout windows (20-180s)
//   - JSON serialization for debuggability
//   - Atomic operations to prevent race conditions on concurrent requests
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// State represents the current position in the USSD menu tree.
type State string

const (
	StateInit              State = "INIT"
	StateMainMenu          State = "MAIN_MENU"
	StateCheckBalance      State = "CHECK_BALANCE"
	StateWithdraw          State = "WITHDRAW"
	StateWithdrawAmount    State = "WITHDRAW_AMOUNT"
	StateWithdrawConfirm   State = "WITHDRAW_CONFIRM"
	StateWithdrawPIN       State = "WITHDRAW_PIN"
	StateEarnings          State = "EARNINGS"
	StateEarningsDaily     State = "EARNINGS_DAILY"
	StateEarningsWeekly    State = "EARNINGS_WEEKLY"
	StateEarningsMonthly   State = "EARNINGS_MONTHLY"
	StateLastPayment       State = "LAST_PAYMENT"
	StateLoanStatus        State = "LOAN_STATUS"
	StateLoanApply         State = "LOAN_APPLY"
	StateLoanAmount        State = "LOAN_AMOUNT"
	StateLoanTenure        State = "LOAN_TENURE"
	StateLoanConfirm       State = "LOAN_CONFIRM"
	StateRegister          State = "REGISTER"
	StateRegisterName      State = "REGISTER_NAME"
	StateRegisterNationalID State = "REGISTER_NATIONAL_ID"
	StateRegisterRole      State = "REGISTER_ROLE"
	StateRegisterConfirm   State = "REGISTER_CONFIRM"
	StateLanguageSelect    State = "LANGUAGE_SELECT"
	StateEnd               State = "END"
)

// Data holds the minimal session payload stored in Redis.
// Keep this struct as small as possible — it's serialized on every request.
type Data struct {
	// Session identity
	SessionID string `json:"sid"`
	MSISDN    string `json:"msisdn"`
	ServiceCode string `json:"svc"`

	// FSM state
	CurrentState State  `json:"state"`
	PreviousState State `json:"prev_state,omitempty"`

	// Collected user inputs (accumulated during multi-step flows)
	Inputs map[string]string `json:"inputs,omitempty"`

	// User identity (populated after authentication/lookup)
	CrewMemberID string `json:"crew_id,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	Language     string `json:"lang"`

	// Tracking
	StepCount   int       `json:"steps"`
	CreatedAt   time.Time `json:"created_at"`
	LastInputAt time.Time `json:"last_input_at"`

	// Idempotency: hash of the last processed request to detect telco retries
	LastRequestHash string `json:"last_hash,omitempty"`
}

// SetInput stores a named input value in the session.
func (d *Data) SetInput(key, value string) {
	if d.Inputs == nil {
		d.Inputs = make(map[string]string)
	}
	d.Inputs[key] = value
}

// GetInput retrieves a named input value from the session.
func (d *Data) GetInput(key string) string {
	if d.Inputs == nil {
		return ""
	}
	return d.Inputs[key]
}

// ClearInputs removes all stored inputs.
func (d *Data) ClearInputs() {
	d.Inputs = nil
}

// Store manages USSD session persistence in Redis.
type Store struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// NewStore creates a new session store backed by Redis.
func NewStore(client *redis.Client, prefix string, ttl time.Duration) *Store {
	return &Store{
		client: client,
		prefix: prefix,
		ttl:    ttl,
	}
}

// key builds the Redis key for a session.
func (s *Store) key(sessionID string) string {
	return s.prefix + sessionID
}

// Get retrieves a session by ID. Returns nil if not found or expired.
func (s *Store) Get(ctx context.Context, sessionID string) (*Data, error) {
	raw, err := s.client.Get(ctx, s.key(sessionID)).Bytes()
	if err == redis.Nil {
		return nil, nil // Session not found / expired
	}
	if err != nil {
		return nil, fmt.Errorf("session get: %w", err)
	}

	var data Data
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("session unmarshal: %w", err)
	}

	return &data, nil
}

// Save persists a session to Redis with TTL refresh.
// This is called on every USSD request to extend the session lifetime.
func (s *Store) Save(ctx context.Context, data *Data) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("session marshal: %w", err)
	}

	if err := s.client.Set(ctx, s.key(data.SessionID), raw, s.ttl).Err(); err != nil {
		return fmt.Errorf("session save: %w", err)
	}

	return nil
}

// Delete removes a session from Redis (called on END or timeout).
func (s *Store) Delete(ctx context.Context, sessionID string) error {
	if err := s.client.Del(ctx, s.key(sessionID)).Err(); err != nil {
		return fmt.Errorf("session delete: %w", err)
	}
	return nil
}

// Exists checks if a session exists without deserializing it.
func (s *Store) Exists(ctx context.Context, sessionID string) (bool, error) {
	n, err := s.client.Exists(ctx, s.key(sessionID)).Result()
	if err != nil {
		return false, fmt.Errorf("session exists: %w", err)
	}
	return n > 0, nil
}

// Touch extends the TTL of an existing session without modifying its data.
func (s *Store) Touch(ctx context.Context, sessionID string) error {
	if err := s.client.Expire(ctx, s.key(sessionID), s.ttl).Err(); err != nil {
		return fmt.Errorf("session touch: %w", err)
	}
	return nil
}

// ActiveCount returns the approximate number of active sessions.
// Uses SCAN to avoid blocking Redis with KEYS.
func (s *Store) ActiveCount(ctx context.Context) (int64, error) {
	var count int64
	var cursor uint64
	for {
		keys, nextCursor, err := s.client.Scan(ctx, cursor, s.prefix+"*", 1000).Result()
		if err != nil {
			return 0, fmt.Errorf("session count scan: %w", err)
		}
		count += int64(len(keys))
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return count, nil
}
