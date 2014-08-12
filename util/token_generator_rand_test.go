package util_test

import (
	"testing"

	"github.com/opentarock/service-user-management/util"
	"github.com/stretchr/testify/assert"
)

func TestGeneratedTokenIsOfCorrectLength(t *testing.T) {
	tokenGenerator := util.NewRandTokenGenerator()
	token, err := tokenGenerator.Generate(100)
	assert.Nil(t, err)
	assert.Equal(t, 100, len(token))
}

func TestGeneratedTokenLookRandom(t *testing.T) {
	tokenGenerator := util.NewRandTokenGenerator()
	token1, err1 := tokenGenerator.Generate(10)
	assert.Nil(t, err1)
	token2, err2 := tokenGenerator.Generate(10)
	assert.Nil(t, err2)
	assert.NotEqual(t, token1, token2)
}

func TestDifferentTokenGeneratorInstancesProduceDifferentTokens(t *testing.T) {
	tokenGenerator1 := util.NewRandTokenGenerator()
	tokenGenerator2 := util.NewRandTokenGenerator()

	token1, err1 := tokenGenerator1.Generate(100)
	token2, err2 := tokenGenerator2.Generate(100)
	assert.Nil(t, err1)
	assert.Nil(t, err2)
	assert.NotEqual(t, token1, token2)
}
