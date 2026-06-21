package app

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
)

const csrfTokenLen = 32 // 32 bytes = 64 hex chars

// generateCSRFToken generates a cryptographically random token for CSRF
// protection of edit API endpoints.
func generateCSRFToken() string {
	b := make([]byte, csrfTokenLen)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("csrf: failed to generate random token: %v", err))
	}
	return hex.EncodeToString(b)
}

// verifyCSRFToken compares a candidate token against the server's stored
// token using constant-time comparison to prevent timing attacks.
func verifyCSRFToken(stored, candidate string) bool {
	if stored == "" || candidate == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(candidate)) == 1
}
