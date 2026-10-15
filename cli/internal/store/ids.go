package store

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func NewHashID(prefix string) (string, error) {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", err
	}
	sum := sha256.Sum256(nonce[:])
	return prefix + hex.EncodeToString(sum[:4]), nil
}
