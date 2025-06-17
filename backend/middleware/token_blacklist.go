package middleware

import (
	"sync"
)

// Use sync.Map for thread-safe operations
var blacklistedTokens sync.Map

// AddTokenToBlacklist adds a token to the blacklist
func AddTokenToBlacklist(token string) {
	blacklistedTokens.Store(token, true)
}

// IsTokenBlacklisted checks if a token is blacklisted
func IsTokenBlacklisted(token string) bool {
	_, found := blacklistedTokens.Load(token)
	return found
}
