package repository

import (
	"database/sql"
	"time"

	"github.com/opentarock/service-api/go/proto_oauth2"
	"github.com/opentarock/service-api/go/proto_user"
	"github.com/opentarock/service-user-management/util"
)

type accessTokenRepositoryPostgres struct {
	db         *sql.DB
	statements map[string]*sql.Stmt
}

func NewAccessTokenRepositoryPostgres(db *sql.DB) *accessTokenRepositoryPostgres {
	repo := &accessTokenRepositoryPostgres{
		db:         db,
		statements: make(map[string]*sql.Stmt),
	}
	util.Prepare(db, repo.statements, "save_access_token",
		`INSERT INTO access_tokens (access_token, client_id, user_id, token_type, expires_in, expires_on, refresh_token)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`)

	return repo
}

func (r *accessTokenRepositoryPostgres) Save(
	user *proto_user.User,
	client *proto_oauth2.Client,
	accessToken *proto_oauth2.AccessToken) error {

	expiresOn := time.Now().Add(time.Duration(accessToken.GetExpiresIn()) * time.Second)
	_, err := util.Exec(r.statements, "save_access_token",
		accessToken.GetAccessToken(),
		client.GetId(), user.GetId(),
		accessToken.GetTokenType(),
		accessToken.GetExpiresIn(),
		expiresOn,
		accessToken.RefreshToken)
	return err
}
