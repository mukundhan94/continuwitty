package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultSessionCookieName is the default cookie used for session state.
	DefaultSessionCookieName = "session"
	defaultSessionTTL        = 24 * time.Hour
)

var (
	errInvalidSessionManager = errors.New("invalid session manager configuration")
	errInvalidSessionToken   = errors.New("invalid session token")
)

// SessionUser stores authenticated user identity in session state.
type SessionUser struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// SessionState stores cookie-backed session values.
type SessionState struct {
	User      *SessionUser `json:"user,omitempty"`
	CSRFToken string       `json:"csrf_token,omitempty"`
	IssuedAt  int64        `json:"issued_at,omitempty"`
}

// SessionManagerOptions configures cookie/session lifecycle behavior.
type SessionManagerOptions struct {
	CookieName string
	TTL        time.Duration
	Now        func() time.Time
}

// SessionManager handles signed session cookie encoding/decoding.
type SessionManager struct {
	secret     []byte
	cookieName string
	ttl        time.Duration
	now        func() time.Time
}

// NewSessionManager creates a session manager for a signing secret and cookie name.
func NewSessionManager(secret, cookieName string) (*SessionManager, error) {
	return NewSessionManagerWithOptions(
		secret,
		SessionManagerOptions{
			CookieName: cookieName,
			TTL:        defaultSessionTTL,
		},
	)
}

// NewSessionManagerWithOptions creates a session manager with lifecycle options.
func NewSessionManagerWithOptions(secret string, options SessionManagerOptions) (*SessionManager, error) {
	secretValue := strings.TrimSpace(secret)
	if secretValue == "" {
		return nil, errInvalidSessionManager
	}

	cookieValue := strings.TrimSpace(options.CookieName)
	if cookieValue == "" {
		cookieValue = DefaultSessionCookieName
	}
	ttl := options.TTL
	if ttl <= 0 {
		ttl = defaultSessionTTL
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &SessionManager{
		secret:     []byte(secretValue),
		cookieName: cookieValue,
		ttl:        ttl,
		now:        now,
	}, nil
}

// CookieName returns the configured cookie name.
func (manager *SessionManager) CookieName() string {
	if manager == nil {
		return DefaultSessionCookieName
	}
	return manager.cookieName
}

// CookieTTL returns the configured TTL for issued session cookies.
func (manager *SessionManager) CookieTTL() time.Duration {
	if manager == nil {
		return defaultSessionTTL
	}
	return manager.ttl
}

// Encode serializes and signs session state to a cookie token.
func (manager *SessionManager) Encode(state SessionState) (string, error) {
	if manager == nil || len(manager.secret) == 0 {
		return "", errInvalidSessionManager
	}
	if state.IssuedAt == 0 {
		state.IssuedAt = manager.now().Unix()
	}

	payloadBytes, err := json.Marshal(state)
	if err != nil {
		return "", errInvalidSessionToken
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signature := manager.sign(payload)
	return payload + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

// Decode validates and parses a signed session token.
func (manager *SessionManager) Decode(token string) (SessionState, error) {
	if manager == nil || len(manager.secret) == 0 {
		return SessionState{}, errInvalidSessionManager
	}
	parts := strings.SplitN(strings.TrimSpace(token), ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return SessionState{}, errInvalidSessionToken
	}

	expectedSignature := manager.sign(parts[0])
	receivedSignature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(receivedSignature, expectedSignature) {
		return SessionState{}, errInvalidSessionToken
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return SessionState{}, errInvalidSessionToken
	}
	var state SessionState
	if err := json.Unmarshal(payloadBytes, &state); err != nil {
		return SessionState{}, errInvalidSessionToken
	}
	if manager.ttl > 0 {
		if state.IssuedAt <= 0 {
			return SessionState{}, errInvalidSessionToken
		}
		expiresAt := time.Unix(state.IssuedAt, 0).Add(manager.ttl)
		if manager.now().After(expiresAt) {
			return SessionState{}, errInvalidSessionToken
		}
	}
	return state, nil
}

// DecodeRequest extracts and decodes session state from request cookies.
func (manager *SessionManager) DecodeRequest(request *http.Request) (SessionState, error) {
	if request == nil {
		return SessionState{}, errInvalidSessionToken
	}
	cookie, err := request.Cookie(manager.CookieName())
	if err != nil {
		return SessionState{}, errInvalidSessionToken
	}
	return manager.Decode(cookie.Value)
}

func (manager *SessionManager) sign(payload string) []byte {
	hash := hmac.New(sha256.New, manager.secret)
	_, _ = hash.Write([]byte(payload))
	return hash.Sum(nil)
}
