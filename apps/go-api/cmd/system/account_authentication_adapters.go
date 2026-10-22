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
		if err := json.Unmarshal(value, &record); err != nil {
			return "", "", fmt.Errorf("decode existing magic code: %w", err)
		}
		if record.CurrentAttempt > 2 {
			exists, err := issuer.lookup(ctx, email)
			if err != nil {
				return "", "", fmt.Errorf("check magic-code account: %w", err)
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

type djangoPasswordResetTokenIssuer struct {
	secret string
	now    func() time.Time
}

func newDjangoPasswordResetTokenIssuer(secret string) (*djangoPasswordResetTokenIssuer, error) {
	if secret == "" {
		return nil, errors.New("Django secret key is required for password reset tokens")
	}
	return &djangoPasswordResetTokenIssuer{secret: secret, now: time.Now}, nil
}

func (issuer *djangoPasswordResetTokenIssuer) Issue(_ context.Context, user store.User) (string, string, error) {
	if user.ID == "" {
		return "", "", errors.New("password reset user id is required")
	}
	uid := base64.RawURLEncoding.EncodeToString([]byte(user.ID))
	timestamp := issuer.now().UTC().Unix() - djangoPasswordResetEpoch.Unix()
	if timestamp < 0 {
		return "", "", errors.New("password reset timestamp predates Django epoch")
	}
	loginTimestamp := ""
	if user.LastLogin != nil {
		loginTimestamp = user.LastLogin.UTC().Truncate(time.Second).Format("2006-01-02 15:04:05")
	}
	return uid, djangoBase36(timestamp) + "-" + issuer.hash(user, loginTimestamp, timestamp), nil
}

func (issuer *djangoPasswordResetTokenIssuer) Verify(_ context.Context, user store.User, uidb64, token string) (bool, error) {
	decodedID, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(uidb64, "="))
	if err != nil || string(decodedID) != user.ID {
		return false, nil
	}
	parts := strings.Split(token, "-")
	if len(parts) != 2 || len(parts[1]) != 32 {
		return false, nil
	}
	timestamp, err := strconv.ParseInt(parts[0], 36, 64)
	if err != nil || timestamp < 0 {
		return false, nil
	}
	issuedAt := djangoPasswordResetEpoch.Add(time.Duration(timestamp) * time.Second)
	age := issuer.now().UTC().Sub(issuedAt)
	if age < 0 || age > 3*24*time.Hour {
		return false, nil
	}
	loginTimestamp := ""
	if user.LastLogin != nil {
		loginTimestamp = user.LastLogin.UTC().Truncate(time.Second).Format("2006-01-02 15:04:05")
	}
	expected := djangoBase36(timestamp) + "-" + issuer.hash(user, loginTimestamp, timestamp)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(token)) == 1, nil
}

func (issuer *djangoPasswordResetTokenIssuer) hash(user store.User, loginTimestamp string, timestamp int64) string {
	value := user.ID + user.Password + loginTimestamp + strconv.FormatInt(timestamp, 10) + user.Email
	keySalt := "django.contrib.auth.tokens.PasswordResetTokenGenerator"
	keyHash := sha256.Sum256([]byte(keySalt + issuer.secret))
	mac := hmac.New(sha256.New, keyHash[:])
	_, _ = mac.Write([]byte(value))
	digest := hex.EncodeToString(mac.Sum(nil))
	shortDigest := make([]byte, 0, len(digest)/2)
	for index := 0; index < len(digest); index += 2 {
		shortDigest = append(shortDigest, digest[index])
	}
	return string(shortDigest)
}

var djangoPasswordResetEpoch = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)

func djangoBase36(value int64) string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	if value == 0 {
		return "0"
	}
	var output [16]byte
	i := len(output)
	for value > 0 {
		i--
		output[i] = alphabet[value%36]
		value /= 36
	}
	return string(output[i:])
}
