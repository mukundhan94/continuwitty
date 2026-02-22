package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

const (
	// DefaultSessionCookieName is the default cookie used for session state.
	DefaultSessionCookieName = "session"
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
}

// SessionManager handles signed session cookie encoding/decoding.
type SessionManager struct {
	secret     []byte
	cookieName string
}

// NewSessionManager creates a session manager for a signing secret and cookie name.
func NewSessionManager(secret, cookieName string) (*SessionManager, error) {
	secretValue := strings.TrimSpace(secret)
	if secretValue == "" {
		return nil, errInvalidSessionManager
	}

	cookieValue := strings.TrimSpace(cookieName)
	if cookieValue == "" {
		cookieValue = DefaultSessionCookieName
	}
	return &SessionManager{
		secret:     []byte(secretValue),
		cookieName: cookieValue,
	}, nil
}

// CookieName returns the configured cookie name.
func (manager *SessionManager) CookieName() string {
	if manager == nil {
		return DefaultSessionCookieName
	}
	return manager.cookieName
}

// Encode serializes and signs session state to a cookie token.
func (manager *SessionManager) Encode(state SessionState) (string, error) {
	if manager == nil || len(manager.secret) == 0 {
		return "", errInvalidSessionManager
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
