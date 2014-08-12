package util_test

import (
	"testing"

	"github.com/opentarock/service-user-management/util"
	"github.com/stretchr/testify/assert"
)

func TestPasswordHashedWithDifferentSaltsHasDifferentHash(t *testing.T) {
	hasher := util.NewPBKDF2PasswordHasher()
	hash1 := hasher.Hash("pass", "salt1")
	hash2 := hasher.Hash("pass", "salt2")
	assert.NotEqual(t, hash1, hash2)
}

func TestDifferentPasswordsWithTheSameSaltHaveDifferentHashes(t *testing.T) {
	hasher := util.NewPBKDF2PasswordHasher()
	hash1 := hasher.Hash("pass1", "salt")
	hash2 := hasher.Hash("pass2", "salt")
	assert.NotEqual(t, hash1, hash2)
}

func TestPasswordHashWithTheSameSaltIsAlwaysTheSame(t *testing.T) {
	hasher1 := util.NewPBKDF2PasswordHasher()
	hasher2 := util.NewPBKDF2PasswordHasher()
	hash1 := hasher1.Hash("pass", "salt")
	hash2 := hasher2.Hash("pass", "salt")
	assert.Equal(t, hash1, hash2)
}
