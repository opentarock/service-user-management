package nnservice_test

import (
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	nmsg "github.com/op/go-nanomsg"
	"github.com/opentarock/service-user-management/nnservice"
)

func TestHandlerIsUsed(t *testing.T) {
	req, err := nmsg.NewReqSocket()
	assert.Nil(t, err)
	repService := nnservice.NewRepService("tcp://*:9000")
	req.Connect("tcp://localhost:9000")
	called := false
	repService.AddHandler(1,
		nnservice.MessageHandlerFunc(func(data []byte) []byte {
			called = true
			return []byte{}
		}))
	go func() {
		repService.Start()
	}()

	req.Send([]byte{1}, 0)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, called, true)
	defer repService.Close()
}

func TestOnlyLastAddedHandlerForTypeIsUsed(t *testing.T) {
	req, err := nmsg.NewReqSocket()
	assert.Nil(t, err)
	repService := nnservice.NewRepService("tcp://*:9000")
	req.Connect("tcp://localhost:9000")
	calledFirst := false
	repService.AddHandler(1,
		nnservice.MessageHandlerFunc(func(data []byte) []byte {
			calledFirst = true
			return []byte{}
		}))
	calledSecond := false
	repService.AddHandler(1,
		nnservice.MessageHandlerFunc(func(data []byte) []byte {
			calledSecond = true
			return []byte{}
		}))
	go func() {
		repService.Start()
	}()

	req.Send([]byte{1}, 0)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, calledFirst, false, "First handler should be owerwritten")
	assert.Equal(t, calledSecond, true)
	defer repService.Close()
}
