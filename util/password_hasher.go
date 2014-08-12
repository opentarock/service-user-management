package util

type PasswordHasher interface {
	Hash(password, salt string) []byte
}
