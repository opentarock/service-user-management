package nnservice

type MessageHandler interface {
	HandleMessage(data []byte) []byte
}

type MessageHandlerFunc func(data []byte) []byte

func (f MessageHandlerFunc) HandleMessage(data []byte) []byte {
	return f(data)
}
