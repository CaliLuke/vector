package apikey

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type throttleCacheStub struct {
	value  []byte
	found  bool
	key    string
	ttl    time.Duration
	getErr error
	setErr error
}

func (cache *throttleCacheStub) Get(_ context.Context, key string) ([]byte, bool, error) {
	cache.key = key
	if cache.getErr != nil {
		return nil, false, cache.getErr
	}
	return cache.value, cache.found, nil
}

func (cache *throttleCacheStub) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	cache.key, cache.value, cache.ttl, cache.found = key, value, ttl, true
	return cache.setErr
}

func TestThrottleSharesDjangoCacheHistoryAndHeaders(t *testing.T) {
	cache := &throttleCacheStub{}
	throttle, err := NewThrottle(cache, "2/min")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	throttle.now = func() time.Time { return now }

	first, err := throttle.Allow(context.Background(), "secret-token")
	if err != nil {
		t.Fatal(err)
	}
	if !first.Allowed || first.Remaining != "1" || first.Reset != now.Unix()+60 {
		t.Fatalf("first rate=%+v", first)
	}
	if cache.key != ":1:api_key:secret-token" || cache.ttl != time.Minute {
		t.Fatalf("cache key=%q ttl=%s", cache.key, cache.ttl)
	}

	second, err := throttle.Allow(context.Background(), "secret-token")
	if err != nil {
		t.Fatal(err)
	}
	if !second.Allowed || second.Remaining != "0" {
		t.Fatalf("second rate=%+v", second)
	}
	third, err := throttle.Allow(context.Background(), "secret-token")
	if err != nil {
		t.Fatal(err)
	}
	if third.Allowed || third.Wait != time.Minute {
		t.Fatalf("third rate=%+v", third)
	}
}

func TestThrottleAllowsExactDjangoCacheKey(t *testing.T) {
	cache := &throttleCacheStub{}
	throttle, err := NewThrottle(cache, "5/minute")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	throttle.now = func() time.Time { return now }

	rate, err := throttle.AllowCacheKey(context.Background(), ":1:throttle_asset_22222222-2222-4222-8222-222222222222", "asset-ID")
	if err != nil {
		t.Fatal(err)
	}
	if !rate.Allowed || rate.Remaining != "4" || cache.key != ":1:throttle_asset_22222222-2222-4222-8222-222222222222" || cache.ttl != time.Minute {
		t.Fatalf("rate=%+v cache key=%q ttl=%s", rate, cache.key, cache.ttl)
	}
}

func TestThrottleReturnsCacheFailures(t *testing.T) {
	readFailure := &throttleCacheStub{getErr: errors.New("read unavailable")}
	throttle, err := NewThrottle(readFailure, "5/minute")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := throttle.AllowCacheKey(context.Background(), ":1:throttle_asset_asset-id", "asset-ID"); err == nil {
		t.Fatal("AllowCacheKey succeeded after cache read failure")
	}

	writeFailure := &throttleCacheStub{setErr: errors.New("write unavailable")}
	throttle, err = NewThrottle(writeFailure, "5/minute")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := throttle.AllowCacheKey(context.Background(), ":1:throttle_asset_asset-id", "asset-ID"); err == nil {
		t.Fatal("AllowCacheKey succeeded after cache write failure")
	}
}

func TestThrottleRejectsCorruptHistoryWithoutOverwritingIt(t *testing.T) {
	corrupt := []byte{0x95, 0x00} // A truncated frame in the persisted history.
	cache := &throttleCacheStub{value: bytes.Clone(corrupt), found: true}
	throttle, err := NewThrottle(cache, "5/minute")
	if err != nil {
		t.Fatal(err)
	}
	rate, err := throttle.Allow(context.Background(), "token")
	if err == nil || !strings.Contains(err.Error(), "decode Django API-key throttle: truncated pickle frame") {
		t.Fatalf("corrupt cache error=%v", err)
	}
	if rate.Allowed || !bytes.Equal(cache.value, corrupt) || cache.ttl != 0 {
		t.Fatalf("corrupt history granted access or was overwritten: rate=%+v cache=%+v", rate, cache)
	}
}

func TestThrottleExpiresOldestDjangoPickleHistory(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 1, 0, 0, time.UTC)
	cache := &throttleCacheStub{value: encodeThrottlePickle([]float64{float64(now.Add(-30 * time.Second).Unix()), float64(now.Add(-61 * time.Second).Unix())}), found: true}
	throttle, _ := NewThrottle(cache, "2/min")
	throttle.now = func() time.Time { return now }

	rate, err := throttle.Allow(context.Background(), "token")
	if err != nil {
		t.Fatal(err)
	}
	if !rate.Allowed || rate.Remaining != "0" {
		t.Fatalf("rate=%+v", rate)
	}
	history, err := decodeThrottlePickle(cache.value)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || history[0] != float64(now.Unix()) {
		t.Fatalf("history=%v", history)
	}
}

func TestThrottleOmitsWaitForOverfullDjangoHistory(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 1, 0, 0, time.UTC)
	cache := &throttleCacheStub{value: encodeThrottlePickle([]float64{
		float64(now.Add(-10 * time.Second).Unix()),
		float64(now.Add(-20 * time.Second).Unix()),
	}), found: true}
	throttle, err := NewThrottle(cache, "1/minute")
	if err != nil {
		t.Fatal(err)
	}
	throttle.now = func() time.Time { return now }
	rate, err := throttle.Allow(context.Background(), "secret-token")
	if err != nil {
		t.Fatal(err)
	}
	if rate.Allowed || rate.Wait != 0 {
		t.Fatalf("rate=%+v", rate)
	}
}

func TestThrottleZeroRateRejectsWithDjangoWait(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 1, 0, 0, time.UTC)
	throttle, err := NewThrottle(&throttleCacheStub{}, "0/minute")
	if err != nil {
		t.Fatal(err)
	}
	throttle.now = func() time.Time { return now }
	rate, err := throttle.Allow(context.Background(), "secret-token")
	if err != nil {
		t.Fatal(err)
	}
	if rate.Allowed || rate.Wait != time.Minute {
		t.Fatalf("rate=%+v", rate)
	}
}
func TestThrottleAllowsDjangoArbitraryPrecisionRate(t *testing.T) {
	throttle, err := NewThrottle(&throttleCacheStub{}, "9223372036854775808/minute")
	if err != nil {
		t.Fatal(err)
	}
	rate, err := throttle.Allow(context.Background(), "secret-token")
	if err != nil {
		t.Fatal(err)
	}
	if !rate.Allowed || rate.Remaining != "9223372036854775807" {
		t.Fatalf("rate=%+v", rate)
	}
}
func TestThrottleRateParserMatchesDRFPeriodPrefixes(t *testing.T) {
	for _, testCase := range []struct {
		rate   string
		window time.Duration
	}{
		{rate: "1/seconds", window: time.Second},
		{rate: "2/minutes", window: time.Minute},
		{rate: "3/hours", window: time.Hour},
		{rate: "4/days", window: 24 * time.Hour},
	} {
		_, window, err := parseRate(testCase.rate)
		if err != nil || window != testCase.window {
			t.Errorf("parseRate(%q) = %s, %v", testCase.rate, window, err)
		}
	}
	count, _, err := parseRate(" ٠_٢/minute")
	if err != nil || count.String() != "2" {
		t.Fatalf("parseRate accepted count=%v err=%v, want Python integer 2", count, err)
	}
	count, _, err = parseRate("0/minute")
	if err != nil || count.Sign() != 0 {
		t.Fatalf("parseRate zero count=%v err=%v", count, err)
	}
	count, _, err = parseRate("9223372036854775808/minute")
	if err != nil || count.String() != "9223372036854775808" {
		t.Fatalf("parseRate large count=%v err=%v", count, err)
	}
	for _, rate := range []string{"", "2/minute/extra"} {
		if _, _, err := parseRate(rate); err == nil {
			t.Errorf("parseRate(%q) succeeded", rate)
		}
	}
	if _, _, err := parseRate("1/Minutes"); err == nil {
		t.Fatal("parseRate accepted the uppercase period that DRF rejects")
	}
}
