package nnservice

import (
	"log"

	_ "github.com/lib/pq"
	nmsg "github.com/op/go-nanomsg"
	"github.com/opentarock/service-user-management/util/logutil"
)

type RepService struct {
	Address         string
	socket          *nmsg.RepSocket
	messageHandlers map[int]MessageHandler
}

func NewRepService(bind string) *RepService {
	return &RepService{
		Address:         bind,
		messageHandlers: make(map[int]MessageHandler),
	}
}

func (s *RepService) AddHandler(messageId int, handler MessageHandler) {
	log.Printf("Adding handler for: %d", messageId)
	s.messageHandlers[messageId] = handler
}

func (s *RepService) Start() {
	socket, err := nmsg.NewRepSocket()
	logutil.ErrorFatal("Error creating response socket", err)
	s.socket = socket

	endpoint, err := socket.Bind(s.Address)
	logutil.ErrorFatal("Error binding socket", err)
	log.Printf("Bound to endpoint: %s", endpoint.Address)

	for {
		recvData, err := socket.Recv(0)
		logutil.ErrorFatal("Error receiving message", err)
		if len(recvData) < 1 {
			log.Printf("Unexpected empty message")
		}
		if handler, ok := s.messageHandlers[int(recvData[0])]; ok {
			responseData := handler.HandleMessage(recvData[1:])
			socket.Send(responseData, 0)
		} else {
			log.Printf("Unknown message type: %d", recvData[0])
		}
	}
}

func (s *RepService) Close() {
	s.socket.Close()
}
