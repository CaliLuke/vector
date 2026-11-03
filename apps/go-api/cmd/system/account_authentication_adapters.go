package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	systemapi "github.com/CaliLuke/go-plane"
	store "github.com/CaliLuke/go-plane/internal/accountauthentication"
	"github.com/redis/go-redis/v9"
)

const magicCodeTTL = 10 * time.Minute

type redisMagicCodeIssuer struct {
	client redis.UniversalClient
	lookup func(context.Context, string) (bool, error)
}

func newRedisMagicCodeIssuer(client redis.UniversalClient, lookup func(context.Context, string) (bool, error)) (*redisMagicCodeIssuer, error) {
	if client == nil || lookup == nil {
		return nil, errors.New("magic-code issuer dependencies are required")
	}
	return &redisMagicCodeIssuer{client: client, lookup: lookup}, nil
}

func (issuer *redisMagicCodeIssuer) Issue(ctx context.Context, email string) (string, string, error) {
	key := "magic_" + email
	value, err := issuer.client.Get(ctx, key).Bytes()
	if err != nil && !errors.Is(err, redis.Nil) {
		return "", "", fmt.Errorf("read existing magic code: %w", err)
	}
	currentAttempt := 0
	if err == nil {
		var record struct {
			CurrentAttempt int `json:"current_attempt"`
		}
		if unmarshalErr := json.Unmarshal(value, &record); unmarshalErr != nil {
			return "", "", fmt.Errorf("decode existing magic code: %w", unmarshalErr)
		}
		if record.CurrentAttempt > 2 {
			exists, lookupErr := issuer.lookup(ctx, email)
			if lookupErr != nil {
				return "", "", fmt.Errorf("check magic-code account: %w", lookupErr)
			}
			if exists {
				return "", "", &systemapi.MagicCodeError{Code: 5100, Message: "EMAIL_CODE_ATTEMPT_EXHAUSTED_SIGN_IN", Email: email}
			}
			return "", "", &systemapi.MagicCodeError{Code: 5102, Message: "EMAIL_CODE_ATTEMPT_EXHAUSTED_SIGN_UP", Email: key}
		}
		currentAttempt++
	}
	tokenValue, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", "", fmt.Errorf("generate magic code: %w", err)
	}
	token := strconv.Itoa(int(tokenValue.Int64()) + 100000)
	encoded, err := json.Marshal(struct {
		CurrentAttempt int    `json:"current_attempt"`
		Email          string `json:"email"`
		Token          string `json:"token"`
	}{CurrentAttempt: currentAttempt, Email: email, Token: token})
	if err != nil {
		return "", "", fmt.Errorf("encode magic code: %w", err)
	}
	if err := issuer.client.Set(ctx, key, encoded, magicCodeTTL).Err(); err != nil {
		return "", "", fmt.Errorf("store magic code: %w", err)
	}
	if err := issuer.client.Del(ctx, key+":verify_attempts").Err(); err != nil {
		return "", "", fmt.Errorf("reset magic-code verify attempts: %w", err)
	}
	return key, token, nil
}

type passwordResetTokenIssuer struct {
	secret string
	now    func() time.Time
}

func newPasswordResetTokenIssuer(secret string) (*passwordResetTokenIssuer, error) {
	if secret == "" {
		return nil, errors.New("password reset signing secret is required")
	}
	return &passwordResetTokenIssuer{secret: secret, now: time.Now}, nil
}

func (issuer *passwordResetTokenIssuer) Issue(_ context.Context, user store.User) (string, string, error) {
	if user.ID == "" {
		return "", "", errors.New("password reset user id is required")
	}
	uid := base64.RawURLEncoding.EncodeToString([]byte(user.ID))
	timestamp := strconv.FormatInt(issuer.now().UTC().Unix(), 10)
	return uid, "v1." + timestamp + "." + issuer.hash(user, timestamp), nil
}

func (issuer *passwordResetTokenIssuer) Verify(_ context.Context, user store.User, uid, token string) (bool, error) {
	return issuer.valid(user, uid, token), nil
}

func (issuer *passwordResetTokenIssuer) valid(user store.User, uid, token string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(uid)
	if err != nil || string(decoded) != user.ID {
		return false
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != "v1" || len(parts[2]) != sha256.Size*2 {
		return false
	}
	timestamp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || timestamp < 0 || strconv.FormatInt(timestamp, 10) != parts[1] {
		return false
	}
	now := issuer.now().UTC().Unix()
	if timestamp > now || now-timestamp > int64((72*time.Hour)/time.Second) {
		return false
	}
	expected := issuer.hash(user, parts[1])
	return subtle.ConstantTimeCompare([]byte(expected), []byte(parts[2])) == 1
}

func (issuer *passwordResetTokenIssuer) hash(user store.User, timestamp string) string {
	login := ""
	if user.LastLogin != nil {
		login = user.LastLogin.UTC().Format(time.RFC3339Nano)
	}
	// Length-framed JSON strings prevent different account fields from sharing a
	// concatenation. The purpose and version isolate this signing domain.
	value, _ := json.Marshal([]string{"vector.password-reset.v1", user.ID, user.Password, login, timestamp, user.Email})
	mac := hmac.New(sha256.New, []byte(issuer.secret))
	_, _ = mac.Write(value)
	return hex.EncodeToString(mac.Sum(nil))
}
