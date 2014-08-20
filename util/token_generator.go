package util

type TokenGenerator interface {
	Generate(n uint) ([]byte, error)
	GenerateHex(n uint) (string, error)
}
