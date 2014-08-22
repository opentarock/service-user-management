package repository

import (
	"github.com/opentarock/service-api/go/proto_oauth2"
	"github.com/opentarock/service-api/go/proto_user"
)

type AccessTokenRepository interface {
	Save(
		user *proto_user.User,
		client *proto_oauth2.Client,
		accessToken *proto_oauth2.AccessToken,
		parentToken *proto_oauth2.AccessToken) error

	FindUserForToken(accessToken *proto_oauth2.AccessToken) (*proto_user.User, error)
	FindByRefreshToken(client *proto_oauth2.Client, refreshToken string) (*proto_oauth2.AccessToken, error)
}
