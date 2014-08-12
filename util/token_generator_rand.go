package util

import "crypto/rand"

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
