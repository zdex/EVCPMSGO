package security

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

func HashSecretSHA256(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func ConstantTimeEqualHex(aHex, bHex string) bool {
	a, err1 := hex.DecodeString(aHex)
	b, err2 := hex.DecodeString(bHex)
	if err1 != nil || err2 != nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}
