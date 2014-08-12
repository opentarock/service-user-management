package util

import (
	"crypto/sha256"

	"code.google.com/p/go.crypto/pbkdf2"
)

type pbkdf2PasswordHasher struct{}

func NewPBKDF2PasswordHasher() *pbkdf2PasswordHasher {
	return &pbkdf2PasswordHasher{}
}

func (s pbkdf2PasswordHasher) Hash(password, salt string) []byte {
	return pbkdf2.Key([]byte(password), []byte(salt), 4096, 64, sha256.New)
}
