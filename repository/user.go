package repository

import (
	"github.com/opentarock/service-api/go/proto_user"
)

type UserRaw struct {
	User *proto_user.User
	Salt string
}

type UserRepository interface {
	Save(user *proto_user.User) error
	FindById(id uint64) (*proto_user.User, error)
	FindByEmail(email string) (*proto_user.User, error)
	FindByEmailAndPassword(emailAddress, passwordPlain string) (*proto_user.User, error)
	Count() (uint64, error)
}
