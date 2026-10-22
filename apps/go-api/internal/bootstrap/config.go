package bootstrap

import (
	"fmt"

	"github.com/CaliLuke/go-plane/internal/instancecipher"
)

type InstanceConfiguration struct {
	Key         string
	Value       *string
	Category    string
	IsEncrypted bool
}

type configDefinition struct {
	key        string
	defaultVal *string
	category   string
	encrypted  bool
}

func defaultValue(value string) *string { return &value }

// These entries mirror plane.utils.instance_config_variables.core. The
// extended list is currently empty in the source tree.
var instanceConfigDefinitions = []configDefinition{
	{key: "ENABLE_SIGNUP", defaultVal: defaultValue("1"), category: "AUTHENTICATION"},
	{key: "ENABLE_EMAIL_PASSWORD", defaultVal: defaultValue("1"), category: "AUTHENTICATION"},
	{key: "ENABLE_MAGIC_LINK_LOGIN", defaultVal: defaultValue("0"), category: "AUTHENTICATION"},
	{key: "DISABLE_WORKSPACE_CREATION", defaultVal: defaultValue("0"), category: "WORKSPACE_MANAGEMENT"},
	{key: "IS_GOOGLE_ENABLED", defaultVal: defaultValue("0"), category: "GOOGLE"},
	{key: "GOOGLE_CLIENT_ID", category: "GOOGLE"},
	{key: "GOOGLE_CLIENT_SECRET", category: "GOOGLE", encrypted: true},
	{key: "ENABLE_GOOGLE_SYNC", defaultVal: defaultValue("0"), category: "GOOGLE"},
	{key: "IS_GITHUB_ENABLED", defaultVal: defaultValue("0"), category: "GITHUB"},
	{key: "GITHUB_CLIENT_ID", category: "GITHUB"},
	{key: "GITHUB_CLIENT_SECRET", category: "GITHUB", encrypted: true},
	{key: "GITHUB_ORGANIZATION_ID", category: "GITHUB"},
	{key: "ENABLE_GITHUB_SYNC", defaultVal: defaultValue("0"), category: "GITHUB"},
	{key: "IS_GITLAB_ENABLED", defaultVal: defaultValue("0"), category: "GITLAB"},
	{key: "GITLAB_HOST", category: "GITLAB"},
	{key: "GITLAB_CLIENT_ID", category: "GITLAB"},
	{key: "GITLAB_CLIENT_SECRET", category: "GITLAB", encrypted: true},
	{key: "ENABLE_GITLAB_SYNC", defaultVal: defaultValue("0"), category: "GITLAB"},
	{key: "IS_GITEA_ENABLED", defaultVal: defaultValue("0"), category: "GITEA"},
	{key: "GITEA_HOST", category: "GITEA"},
	{key: "GITEA_CLIENT_ID", category: "GITEA"},
	{key: "GITEA_CLIENT_SECRET", category: "GITEA", encrypted: true},
	{key: "ENABLE_GITEA_SYNC", defaultVal: defaultValue("0"), category: "GITEA"},
	{key: "ENABLE_SMTP", defaultVal: defaultValue("0"), category: "SMTP"},
	{key: "EMAIL_HOST", defaultVal: defaultValue(""), category: "SMTP"},
	{key: "EMAIL_HOST_USER", defaultVal: defaultValue(""), category: "SMTP"},
	{key: "EMAIL_HOST_PASSWORD", defaultVal: defaultValue(""), category: "SMTP", encrypted: true},
	{key: "EMAIL_PORT", defaultVal: defaultValue("587"), category: "SMTP"},
	{key: "EMAIL_FROM", defaultVal: defaultValue(""), category: "SMTP"},
	{key: "EMAIL_USE_TLS", defaultVal: defaultValue("1"), category: "SMTP"},
	{key: "EMAIL_USE_SSL", defaultVal: defaultValue("0"), category: "SMTP"},
	{key: "LLM_API_KEY", category: "AI", encrypted: true},
	{key: "LLM_PROVIDER", defaultVal: defaultValue("openai"), category: "AI"},
	{key: "LLM_MODEL", defaultVal: defaultValue("gpt-4o-mini"), category: "AI"},
	{key: "GPT_ENGINE", defaultVal: defaultValue("gpt-3.5-turbo"), category: "AI"},
	{key: "UNSPLASH_ACCESS_KEY", defaultVal: defaultValue(""), category: "UNSPLASH", encrypted: true},
}

func InstanceConfigurations(lookupEnv func(string) (string, bool), cipher *instancecipher.Cipher) ([]InstanceConfiguration, error) {
	if lookupEnv == nil {
		return nil, fmt.Errorf("environment lookup is required")
	}
	if cipher == nil {
		return nil, fmt.Errorf("instance configuration cipher is required")
	}
	configurations := make([]InstanceConfiguration, 0, len(instanceConfigDefinitions))
	for _, definition := range instanceConfigDefinitions {
		value, present := lookupEnv(definition.key)
		if !present {
			if definition.defaultVal == nil {
				if definition.encrypted {
					empty := ""
					value = empty
				} else {
					configurations = append(configurations, InstanceConfiguration{Key: definition.key, Category: definition.category})
					continue
				}
			} else {
				value = *definition.defaultVal
			}
		}
		if definition.encrypted {
			encrypted, err := cipher.Encrypt(value)
			if err != nil {
				return nil, fmt.Errorf("encrypt instance configuration %q: %w", definition.key, err)
			}
			value = encrypted
		}
		valueCopy := value
		configurations = append(configurations, InstanceConfiguration{
			Key: definition.key, Value: &valueCopy, Category: definition.category,
			IsEncrypted: definition.encrypted,
		})
	}
	return configurations, nil
}
