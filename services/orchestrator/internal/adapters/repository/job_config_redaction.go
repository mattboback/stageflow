package db

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

const redactedConfigValue = "[REDACTED]"

func sanitizeTerminalJobConfig(raw string) (string, []string, error) {
	var config map[string]any
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return "", nil, fmt.Errorf("unmarshal terminal job config: %w", err)
	}

	secrets := make([]string, 0)
	sanitizeTerminalConfigMap(config, &secrets)
	secrets = uniqueSensitiveValues(secrets)
	redactKnownConfigSecretsInMap(config, secrets)

	encoded, err := json.Marshal(config)
	if err != nil {
		return "", nil, fmt.Errorf("marshal terminal job config: %w", err)
	}

	return string(encoded), secrets, nil
}

func redactKnownConfigSecretsInMap(value map[string]any, secrets []string) {
	for key, entry := range value {
		value[key] = redactKnownConfigSecretsInValue(entry, secrets)
	}
}

func redactKnownConfigSecretsInValue(value any, secrets []string) any {
	switch typed := value.(type) {
	case string:
		return redactKnownConfigSecrets(typed, secrets)
	case []any:
		for index, entry := range typed {
			typed[index] = redactKnownConfigSecretsInValue(entry, secrets)
		}

		return typed
	case map[string]any:
		redactKnownConfigSecretsInMap(typed, secrets)

		return typed
	default:
		return value
	}
}

func sanitizeTerminalConfigMap(value map[string]any, secrets *[]string) {
	for key, entry := range value {
		switch normalizeConfigKey(key) {
		case "auth":
			collectAuthSecretValues(entry, secrets)
			delete(value, key)
		case "inputvalues":
			value[key] = redactInputValueMap(entry, secrets)
		case "password", "passwd", "username":
			collectStringValues(entry, secrets)

			value[key] = redactedConfigValue
		default:
			sanitizeTerminalConfigValue(entry, secrets)
		}
	}
}

func sanitizeTerminalConfigValue(value any, secrets *[]string) {
	switch typed := value.(type) {
	case map[string]any:
		sanitizeTerminalConfigMap(typed, secrets)
	case []any:
		for _, entry := range typed {
			sanitizeTerminalConfigValue(entry, secrets)
		}
	}
}

func collectAuthSecretValues(value any, secrets *[]string) {
	auth, ok := value.(map[string]any)
	if !ok {
		return
	}

	for key, entry := range auth {
		switch normalizeConfigKey(key) {
		case "contentb64", "password", "passwd", "username":
			collectStringValues(entry, secrets)
		case "steps":
			steps, isSteps := entry.([]any)
			if !isSteps {
				continue
			}

			for _, rawStep := range steps {
				step, isStep := rawStep.(map[string]any)
				if !isStep {
					continue
				}

				actionType, isActionType := step["type"].(string)
				if isActionType && (actionType == "fill" || actionType == "select") {
					collectStringValues(step["value"], secrets)
				}
			}
		}
	}
}

func redactInputValueMap(value any, secrets *[]string) any {
	values, ok := value.(map[string]any)
	if !ok {
		collectStringValues(value, secrets)
		return map[string]any{"redacted": true}
	}

	redacted := make(map[string]any, len(values))
	for key, entry := range values {
		collectStringValues(entry, secrets)

		redacted[key] = redactedConfigValue
	}

	return redacted
}

func collectStringValues(value any, values *[]string) {
	switch typed := value.(type) {
	case string:
		if typed != "" && typed != redactedConfigValue {
			*values = append(*values, typed)
		}
	case []any:
		for _, entry := range typed {
			collectStringValues(entry, values)
		}
	case map[string]any:
		for _, entry := range typed {
			collectStringValues(entry, values)
		}
	}
}

func uniqueSensitiveValues(values []string) []string {
	unique := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value != "" {
			unique[value] = struct{}{}
		}
	}

	out := make([]string, 0, len(unique))
	for value := range unique {
		out = append(out, value)
	}

	sort.Slice(out, func(i, j int) bool {
		return len(out[i]) > len(out[j])
	})

	return out
}

func redactKnownConfigSecrets(value string, secrets []string) string {
	for _, representation := range encodedSecretRepresentations(secrets) {
		value = strings.ReplaceAll(value, representation, redactedConfigValue)
	}

	return value
}

// encodedSecretRepresentations covers the encodings browsers use when a form
// submits secrets in a URL. Failure messages and page URLs can otherwise echo
// a percent- or form-encoded credential even after its raw value is scrubbed.
func encodedSecretRepresentations(secrets []string) []string {
	values := make([]string, 0, len(secrets)*4)
	seen := make(map[string]struct{}, len(secrets)*4)

	for _, secret := range secrets {
		queryEncoded := url.QueryEscape(secret)
		percentEncoded := strings.ReplaceAll(queryEncoded, "+", "%20")
		componentEncoded := encodeBrowserURLValue(secret, false)
		formEncoded := encodeBrowserURLValue(secret, true)

		for _, value := range []string{
			secret,
			queryEncoded,
			percentEncoded,
			url.PathEscape(secret),
			componentEncoded,
			formEncoded,
			lowercasePercentEscapes(queryEncoded),
			lowercasePercentEscapes(percentEncoded),
			lowercasePercentEscapes(componentEncoded),
			lowercasePercentEscapes(formEncoded),
		} {
			if value == "" {
				continue
			}

			if _, exists := seen[value]; exists {
				continue
			}

			seen[value] = struct{}{}
			values = append(values, value)
		}
	}

	sort.Slice(values, func(i, j int) bool { return len(values[i]) > len(values[j]) })

	return values
}

// encodeBrowserURLValue mirrors the two browser encoders used in this path:
// encodeURIComponent when form is false, and WHATWG
// application/x-www-form-urlencoded encoding when form is true.
func encodeBrowserURLValue(value string, form bool) string {
	const upperHex = "0123456789ABCDEF"

	var encoded strings.Builder
	encoded.Grow(len(value))

	for index := range len(value) {
		char := value[index]
		if browserURLByteAllowed(char, form) {
			encoded.WriteByte(char)
			continue
		}

		if form && char == ' ' {
			encoded.WriteByte('+')
			continue
		}

		encoded.WriteByte('%')
		encoded.WriteByte(upperHex[char>>4])
		encoded.WriteByte(upperHex[char&0x0f])
	}

	return encoded.String()
}

func browserURLByteAllowed(value byte, form bool) bool {
	if value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' ||
		value >= '0' && value <= '9' {
		return true
	}

	if form {
		return value == '*' || value == '-' || value == '.' || value == '_'
	}

	switch value {
	case '-', '_', '.', '!', '~', '*', '\'', '(', ')':
		return true
	default:
		return false
	}
}

func lowercasePercentEscapes(value string) string {
	encoded := []byte(value)
	for index := 0; index+2 < len(encoded); index++ {
		if encoded[index] != '%' {
			continue
		}

		encoded[index+1] = lowercaseHexByte(encoded[index+1])
		encoded[index+2] = lowercaseHexByte(encoded[index+2])
		index += 2
	}

	return string(encoded)
}

func lowercaseHexByte(value byte) byte {
	if value >= 'A' && value <= 'F' {
		return value + ('a' - 'A')
	}

	return value
}

func normalizeConfigKey(key string) string {
	replacer := strings.NewReplacer("_", "", "-", "", " ", "")

	return replacer.Replace(strings.ToLower(key))
}
