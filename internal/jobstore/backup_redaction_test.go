package jobstore

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestStorageStateCredentialsRedactedFromBackupMetadata(t *testing.T) {
	state := `{"cookies":[{"name":"session","value":"cookie secret!"}],"origins":[{"origin":"https://example.com","localStorage":[{"name":"token","value":"local token!"}]}]}`
	encoded := base64.StdEncoding.EncodeToString([]byte(state))

	raw, err := json.Marshal(map[string]any{
		"auth":  map[string]any{"content_b64": encoded},
		"pages": []string{"https://example.com/?token=local%20token!"},
		"safe":  "ordinary metadata",
	})
	if err != nil {
		t.Fatal(err)
	}

	config, secrets, err := sanitizeTerminalJobConfig(string(raw))
	if err != nil {
		t.Fatal(err)
	}

	metadata := redactKnownConfigSecrets("failed: cookie+secret%21 / local%20token! / "+encoded, secrets)
	for _, secret := range []string{encoded, "cookie+secret%21", "local%20token!"} {
		if strings.Contains(config+metadata, secret) {
			t.Fatalf("credential survived scrub: %q", secret)
		}
	}

	if !strings.Contains(config, "ordinary metadata") {
		t.Fatal("unrelated metadata lost")
	}
}

func TestKnownCredentialKeysRedactedDeterministically(t *testing.T) {
	const raw = `{"auth":{"mode":"form","steps":[{"type":"fill","value":{"literal":"raw token!"}}]},"scanner_configs":{"axe":{"nested":[{"raw token!":{"raw token!":"first private payload"},"raw%20token!":"second private payload","[REDACTED]":"original conflicting payload","safe":"ordinary metadata"},{"raw+token%21":{"inner":{"raw%20token!":"safe value"}}}]}}}`

	var previous string

	for range 20 {
		config, _, err := sanitizeTerminalJobConfig(raw)
		if err != nil {
			t.Fatal(err)
		}

		for _, forbidden := range []string{
			"raw token!", "raw%20token!", "raw+token%21",
			"first private payload", "second private payload", "original conflicting payload",
		} {
			if strings.Contains(config, forbidden) {
				t.Fatalf("secret key or conflicting value survived: %q", forbidden)
			}
		}

		if !strings.Contains(config, `"[REDACTED]":"[REDACTED]"`) ||
			!strings.Contains(config, `"safe":"ordinary metadata"`) ||
			!strings.Contains(config, `"[REDACTED]":"safe value"`) {
			t.Fatalf("unexpected key scrub result: %s", config)
		}

		if previous != "" && previous != config {
			t.Fatal("key collisions produced nondeterministic JSON")
		}

		previous = config
	}
}

func TestKeyRedactionStableForOverlappingCredentials(t *testing.T) {
	var previous string

	for _, secrets := range [][]string{{"ab", "bc"}, {"bc", "ab"}} {
		value := map[string]any{"abc": "abc"}
		redactKnownConfigSecretsInMap(value, secrets)

		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}

		if previous != "" && previous != string(encoded) {
			t.Fatal("overlapping credentials produced order-dependent output")
		}

		previous = string(encoded)
	}
}
