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

// MaxInputs is the maximum number of named inputs stored in a session.
// This prevents unbounded memory growth from multi-step flows.
// Normal USSD flows use 8-10 keys max; 20 provides generous headroom.
const MaxInputs = 20

// MaxSteps is the maximum number of steps allowed in a single session.
// Prevents runaway sessions from consuming resources indefinitely.
// Normal flows complete in 5-8 steps; 30 allows for retry/back navigation.
const MaxSteps = 30

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
	StateLoanCategory      State = "LOAN_CATEGORY"
	StateLoanAmount        State = "LOAN_AMOUNT"
	StateLoanTenure        State = "LOAN_TENURE"
	StateLoanConfirm       State = "LOAN_CONFIRM"
	StateRegister          State = "REGISTER"
	StateRegisterName      State = "REGISTER_NAME"
	StateRegisterNationalID State = "REGISTER_NATIONAL_ID"
	StateRegisterRole      State = "REGISTER_ROLE"
	StateRegisterPIN       State = "REGISTER_PIN"
	StateRegisterPINConfirm State = "REGISTER_PIN_CONFIRM"
	StateRegisterConfirm   State = "REGISTER_CONFIRM"
	StateMyProfile         State = "MY_PROFILE"
	StateSetPIN            State = "SET_PIN"
	StateSetPINConfirm     State = "SET_PIN_CONFIRM"
	StateChangePIN         State = "CHANGE_PIN"
	StateChangePINNew      State = "CHANGE_PIN_NEW"
	StateChangePINConfirm  State = "CHANGE_PIN_CONFIRM"
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
// Silently drops new entries if MaxInputs is reached (safety bound).
func (d *Data) SetInput(key, value string) {
	if d.Inputs == nil {
		d.Inputs = make(map[string]string, 8) // Pre-size for typical flow
	}
	// Allow overwrites of existing keys (no growth), block new keys beyond cap
	if _, exists := d.Inputs[key]; !exists && len(d.Inputs) >= MaxInputs {
		return
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
	client     *redis.Client
	prefix     string
	ttl        time.Duration
	counterKey string // Atomic counter key for active sessions
}

// NewStore creates a new session store backed by Redis.
func NewStore(client *redis.Client, prefix string, ttl time.Duration) *Store {
	return &Store{
		client:     client,
		prefix:     prefix,
		ttl:        ttl,
		counterKey: prefix + "_active_count",
	}
}

// key builds the Redis key for a session.
func (s *Store) key(sessionID string) string {
	return s.prefix + sessionID
}

// langKey builds the Redis key for a user's language preference (persists across sessions).
func (s *Store) langKey(msisdn string) string {
	return "ussd:lang:" + msisdn
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

// SaveNew persists a brand-new session and increments the atomic active counter.
func (s *Store) SaveNew(ctx context.Context, data *Data) error {
	if err := s.Save(ctx, data); err != nil {
		return err
	}
	s.client.Incr(ctx, s.counterKey) // Best-effort — metric only
	return nil
}

// Delete removes a session from Redis and decrements the active counter.
func (s *Store) Delete(ctx context.Context, sessionID string) error {
	if err := s.client.Del(ctx, s.key(sessionID)).Err(); err != nil {
		return fmt.Errorf("session delete: %w", err)
	}
	s.client.Decr(ctx, s.counterKey) // Best-effort — metric only
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

// ActiveCount returns the approximate number of active sessions using an atomic counter.
// O(1) operation — safe to call from /metrics endpoint at any scale.
func (s *Store) ActiveCount(ctx context.Context) (int64, error) {
	count, err := s.client.Get(ctx, s.counterKey).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("session count: %w", err)
	}
	if count < 0 {
		return 0, nil // Clamp to 0 (can drift negative due to TTL expiry without decrement)
	}
	return count, nil
}

// SaveLanguage persists a user's language preference keyed by phone number.
// This is stored separately from the session so it survives session endings.
func (s *Store) SaveLanguage(ctx context.Context, msisdn, lang string) error {
	if err := s.client.Set(ctx, s.langKey(msisdn), lang, 0).Err(); err != nil {
		return fmt.Errorf("language save: %w", err)
	}
	return nil
}

// GetLanguage retrieves a user's persisted language preference.
// Returns empty string if no preference has been saved.
func (s *Store) GetLanguage(ctx context.Context, msisdn string) (string, error) {
	lang, err := s.client.Get(ctx, s.langKey(msisdn)).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("language get: %w", err)
	}
	return lang, nil
}
