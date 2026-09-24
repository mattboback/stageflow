package main

func envOr(getenv getenvFunc, key, fallback string) string {
	if v := getenv(key); v != "" {
		return v
	}

	return fallback
}
