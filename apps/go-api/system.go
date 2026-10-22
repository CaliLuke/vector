package systemapi

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"math/big"
	"strings"
	"unicode/utf8"

	csrftoken "github.com/CaliLuke/go-plane/gen/csrf_token"
	system "github.com/CaliLuke/go-plane/gen/system"
	"github.com/CaliLuke/loom/security"
	"github.com/jackc/pgx/v5/pgxpool"
)

type systemService struct {
	database *pgxpool.Pool
}

const (
	csrfAllowedChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	csrfSecretLength = 32
	csrfTokenLength  = 64
)

// NewSystem returns the system service implementation.
func NewSystem() system.Service {
	return &systemService{}
}

// NewCSRFToken returns the CSRF token service implementation.
func NewCSRFToken() csrftoken.Service {
	return &systemService{}
}

// NewSystemWithDatabase returns the system service with PostgreSQL readiness checks.
func NewSystemWithDatabase(database *pgxpool.Pool) system.Service {
	return &systemService{database: database}
}

// APIKeyAuth accepts the optional cookie transport in Plane's session contract.
// The CSRF routes do not authorize or personalize their responses.
func (s *systemService) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, nil
}

// NormalizeCSRFSecret validates and normalizes Django's CSRF cookie value.
// Invalid and missing cookies receive a new secret, matching CsrfViewMiddleware.
func NormalizeCSRFSecret(cookieValue string) (string, error) {
	secret := cookieValue
	switch {
	case validCSRFValue(secret, csrfSecretLength):
	case validCSRFValue(secret, csrfTokenLength):
		secret = unmaskCSRFToken(secret)
	default:
		var err error
		secret, err = randomCSRFString(csrfSecretLength)
		if err != nil {
			return "", fmt.Errorf("generate CSRF secret: %w", err)
		}
	}
	return secret, nil
}

// ValidCSRFToken checks Django's cookie and submitted token representations.
// Django accepts either the plain secret or a masked token that unmasks to the
// cookie secret; both values must use its allowed alphabet and length.
func ValidCSRFToken(cookieValue, submitted string) bool {
	secret := cookieValue
	switch {
	case validCSRFValue(secret, csrfSecretLength):
	case validCSRFValue(secret, csrfTokenLength):
		secret = unmaskCSRFToken(secret)
	default:
		return false
	}
	switch {
	case validCSRFValue(submitted, csrfSecretLength):
	case validCSRFValue(submitted, csrfTokenLength):
		submitted = unmaskCSRFToken(submitted)
	default:
		return false
	}
	return subtle.ConstantTimeCompare([]byte(secret), []byte(submitted)) == 1
}

// CSRFTokenFormatReason reports Django's format diagnostic using the same
// alphabet and lengths as token verification. An empty reason means valid format.
func CSRFTokenFormatReason(value string) string {
	length := utf8.RuneCountInString(value)
	if length != csrfSecretLength && length != csrfTokenLength {
		return "has incorrect length"
	}
	if !validCSRFValue(value, len(value)) {
		return "has invalid characters"
	}
	return ""
}

// CsrfToken returns a new mask for the request's stable CSRF secret.
func (s *systemService) CsrfToken(_ context.Context, payload *csrftoken.CsrfTokenPayload) (*csrftoken.CSRFTokenResult, error) {
	secret, err := csrfSecretFromCookie(payload.CsrfCookie)
	if err != nil {
		return nil, err
	}
	token, err := maskCSRFSecret(secret)
	if err != nil {
		return nil, fmt.Errorf("mask CSRF secret: %w", err)
	}
	return &csrftoken.CSRFTokenResult{CsrfToken: token, CsrfCookie: secret}, nil
}

// CsrfTokenOptions describes the representations accepted and returned by the endpoint.
func (s *systemService) CsrfTokenOptions(_ context.Context, _ *csrftoken.CsrfTokenOptionsPayload) (*csrftoken.CSRFTokenOptionsResult, error) {
	return &csrftoken.CSRFTokenOptionsResult{
		Name:        "Csrf Token Endpoint",
		Description: "",
		Renders:     []string{"application/json"},
		Parses:      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
	}, nil
}

// Live reports whether the process can serve requests.
func (s *systemService) Live(ctx context.Context) (*system.LiveResult, error) {
	return &system.LiveResult{Status: "ok"}, nil
}

// Ready reports whether PostgreSQL can execute a query.
func (s *systemService) Ready(ctx context.Context) (*system.ReadyResult, error) {
	if s.database == nil {
		return nil, fmt.Errorf("database is not configured")
	}
	var value int
	if err := s.database.QueryRow(ctx, "SELECT 1").Scan(&value); err != nil {
		return nil, fmt.Errorf("query database readiness: %w", err)
	}
	return &system.ReadyResult{Status: "ok"}, nil
}

func validCSRFValue(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, char := range value {
		if !strings.ContainsRune(csrfAllowedChars, char) {
			return false
		}
	}
	return true
}

func csrfSecretFromCookie(cookie *string) (string, error) {
	if cookie == nil {
		return NormalizeCSRFSecret("")
	}
	return NormalizeCSRFSecret(*cookie)
}

func randomCSRFString(length int) (string, error) {
	result := make([]byte, length)
	limit := big.NewInt(int64(len(csrfAllowedChars)))
	for index := range result {
		choice, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		result[index] = csrfAllowedChars[choice.Int64()]
	}
	return string(result), nil
}

func maskCSRFSecret(secret string) (string, error) {
	mask, err := randomCSRFString(csrfSecretLength)
	if err != nil {
		return "", err
	}
	cipher := make([]byte, csrfSecretLength)
	for index := range cipher {
		secretIndex := strings.IndexByte(csrfAllowedChars, secret[index])
		maskIndex := strings.IndexByte(csrfAllowedChars, mask[index])
		cipher[index] = csrfAllowedChars[(secretIndex+maskIndex)%len(csrfAllowedChars)]
	}
	return mask + string(cipher), nil
}

func unmaskCSRFToken(token string) string {
	mask := token[:csrfSecretLength]
	cipher := token[csrfSecretLength:]
	secret := make([]byte, csrfSecretLength)
	for index := range secret {
		cipherIndex := strings.IndexByte(csrfAllowedChars, cipher[index])
		maskIndex := strings.IndexByte(csrfAllowedChars, mask[index])
		secret[index] = csrfAllowedChars[(cipherIndex-maskIndex+len(csrfAllowedChars))%len(csrfAllowedChars)]
	}
	return string(secret)
}
