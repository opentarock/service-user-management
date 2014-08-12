package util

type TokenGenerator interface {
	Generate(n uint) ([]byte, error)
}
