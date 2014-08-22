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
		`INSERT INTO access_tokens (access_token, client_id, user_id, token_type, expires_in, expires_on, refresh_token, parent_token)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`)

	util.Prepare(db, repo.statements, "find_by_refresh_token",
		`SELECT access_token, token_type, expires_in
		 FROM access_tokens
		 WHERE client_id = $1 AND refresh_token = $2`)

	util.Prepare(db, repo.statements, "find_user_by_token",
		`SELECT id, display_name, email, password
		 FROM users u INNER JOIN access_tokens at
		 ON u.id = at.user_id
		 WHERE at.access_token = $1;`)

	return repo
}

func (r *accessTokenRepositoryPostgres) Save(
	user *proto_user.User,
	client *proto_oauth2.Client,
	accessToken *proto_oauth2.AccessToken,
	parentToken *proto_oauth2.AccessToken) error {

	expiresOn := time.Now().Add(time.Duration(accessToken.GetExpiresIn()) * time.Second)
	var parentTokenId interface{}
	if parentToken != nil {
		parentTokenId = parentToken.GetAccessToken()
	}
	_, err := util.Exec(r.statements, "save_access_token",
		accessToken.GetAccessToken(),
		client.GetId(), user.GetId(),
		accessToken.GetTokenType(),
		accessToken.GetExpiresIn(),
		expiresOn,
		accessToken.RefreshToken,
		parentTokenId)
	return err
}

func (r *accessTokenRepositoryPostgres) FindUserForToken(
	accessToken *proto_oauth2.AccessToken) (*proto_user.User, error) {

	var id uint64
	var displayName, email, password string
	err := util.QueryRow(r.statements, "find_user_by_token", accessToken.GetAccessToken()).Scan(
		&id, &displayName, &email, &password)

	if err != nil {
		return nil, err
	}
	return &proto_user.User{
		Id:          &id,
		DisplayName: &displayName,
		Email:       &email,
		Password:    &password,
	}, nil
}

func (r *accessTokenRepositoryPostgres) FindByRefreshToken(
	client *proto_oauth2.Client, refreshToken string) (*proto_oauth2.AccessToken, error) {

	var accessToken, tokenType string
	var expiresIn uint64

	err := util.QueryRow(r.statements, "find_by_refresh_token", client.GetId(), refreshToken).Scan(
		&accessToken, &tokenType, &expiresIn)

	if err != nil {
		return nil, err
	}
	return &proto_oauth2.AccessToken{
		AccessToken:  &accessToken,
		TokenType:    &tokenType,
		ExpiresIn:    &expiresIn,
		RefreshToken: &refreshToken,
	}, nil
}
