package util

import (
	"crypto/rand"
	"encoding/hex"
)

type randTokenGenerator struct{}

func NewRandTokenGenerator() *randTokenGenerator {
	return &randTokenGenerator{}
}

func (s *randTokenGenerator) Generate(n uint) ([]byte, error) {
	token := make([]byte, n)
	_, err := rand.Read(token)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (s *randTokenGenerator) GenerateHex(n uint) (string, error) {
	token, err := s.Generate(n)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(token), nil
}
