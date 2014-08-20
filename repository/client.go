package repository

import "github.com/opentarock/service-api/go/proto_oauth2"

type ClientRepository interface {
	FindById(clientId string) (*proto_oauth2.Client, error)
}
