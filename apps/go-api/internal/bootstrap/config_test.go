package bootstrap

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/instancecipher"
)

func TestInstanceConfigurationsPreserveDjangoDefaultsAndEncryptSecrets(t *testing.T) {
	cipher, err := instancecipher.New("bootstrap-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	environment := map[string]string{"GOOGLE_CLIENT_SECRET": "google-secret", "ENABLE_SIGNUP": "0"}
	configurations, err := InstanceConfigurations(func(key string) (string, bool) {
		value, ok := environment[key]
		return value, ok
	}, cipher)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(configurations), 36; got != want {
		t.Fatalf("configuration count = %d, want %d", got, want)
	}
	byKey := make(map[string]InstanceConfiguration, len(configurations))
	for _, configuration := range configurations {
		byKey[configuration.Key] = configuration
	}
	assertValue := func(key, want string) {
		t.Helper()
		configuration, ok := byKey[key]
		if !ok || configuration.Value == nil || *configuration.Value != want {
			t.Fatalf("configuration %q = %#v, want %q", key, configuration, want)
		}
	}
	assertValue("ENABLE_SIGNUP", "0")
	assertValue("ENABLE_MAGIC_LINK_LOGIN", "0")
	assertValue("EMAIL_PORT", "587")
	assertValue("LLM_PROVIDER", "openai")
	assertValue("LLM_MODEL", "gpt-4o-mini")
	assertValue("GPT_ENGINE", "gpt-3.5-turbo")
	if byKey["GOOGLE_CLIENT_ID"].Value != nil {
		t.Fatalf("unset optional value should remain SQL NULL, got %#v", byKey["GOOGLE_CLIENT_ID"].Value)
	}
	secret := byKey["GOOGLE_CLIENT_SECRET"]
	if !secret.IsEncrypted || secret.Value == nil {
		t.Fatalf("secret metadata/value = %#v", secret)
	}
	plaintext, err := cipher.Decrypt(*secret.Value)
	if err != nil {
		t.Fatal(err)
	}
	if plaintext != "google-secret" {
		t.Fatalf("decrypted secret = %q, want google-secret", plaintext)
	}
	emptySecret := byKey["LLM_API_KEY"]
	if !emptySecret.IsEncrypted || emptySecret.Value == nil || *emptySecret.Value != "" {
		t.Fatalf("unset encrypted values must match Django's empty string: %#v", emptySecret)
	}
}

func TestValidateDatabaseURL(t *testing.T) {
	for _, value := range []string{"postgres://user:pass@localhost:5432/plane", "postgresql://localhost/plane"} {
		if err := ValidateDatabaseURL(value); err != nil {
			t.Errorf("ValidateDatabaseURL(%q): %v", value, err)
		}
	}
	for _, value := range []string{"", "http://localhost/db", "postgres://"} {
		if err := ValidateDatabaseURL(value); err == nil {
			t.Errorf("ValidateDatabaseURL(%q) succeeded", value)
		}
	}
}
