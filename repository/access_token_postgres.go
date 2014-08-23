package repository

import (
	"database/sql"
	"log"
	"time"

	"github.com/opentarock/service-api/go/proto_oauth2"
	"github.com/opentarock/service-api/go/proto_user"
	"github.com/opentarock/service-user-management/util"
)

type AccessTokenRaw struct {
	Token       *proto_oauth2.AccessToken
	ClientId    string
	UserId      uint64
	ParentToken *string
}

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

	util.Prepare(db, repo.statements, "find_by_token",
		`SELECT token_type, client_id, user_id, expires_in, refresh_token, parent_token
		 FROM access_tokens
		 WHERE access_token = $1 AND expires_on > NOW()`)

	util.Prepare(db, repo.statements, "find_by_refresh_token",
		`SELECT access_token, token_type, expires_in
		 FROM access_tokens
		 WHERE client_id = $1 AND refresh_token = $2`)

	util.Prepare(db, repo.statements, "find_user_by_token",
		`SELECT id, display_name, email, password
		 FROM users u INNER JOIN access_tokens at
		 ON u.id = at.user_id
		 WHERE at.access_token = $1;`)

	util.Prepare(db, repo.statements, "clear_token_parent",
		`UPDATE access_tokens
		 SET parent_token = NULL
		 WHERE access_token = $1`)

	util.Prepare(db, repo.statements, "delete_token_and_parents",
		`WITH RECURSIVE parent_tokens(access_token, parent_token) AS (
		   SELECT access_token, parent_token
		   FROM access_tokens
		   WHERE access_token = $1
		 UNION ALL
		   SELECT at.access_token, at.parent_token
		   FROM parent_tokens pt, access_tokens at
		   WHERE at.access_token = pt.parent_token
	     )
	     DELETE FROM access_tokens
	     WHERE access_token IN (SELECT access_token FROM parent_tokens);`)

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

func (r *accessTokenRepositoryPostgres) DeleteParents(accessToken *AccessTokenRaw) error {
	tx, err := r.db.Begin()
	clearTokenParentStmt := tx.Stmt(r.statements["clear_token_parent"])
	_, err = clearTokenParentStmt.Exec(accessToken.Token.GetAccessToken())
	if err != nil {
		return tryRollback(tx, err)
	}
	deleteTokenAndParentStmt := tx.Stmt(r.statements["delete_token_and_parents"])
	_, err = deleteTokenAndParentStmt.Exec(accessToken.ParentToken)
	if err != nil {
		return tryRollback(tx, err)
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	accessToken.ParentToken = nil
	return nil
}

func tryRollback(tx *sql.Tx, cause error) error {
	err := tx.Rollback()
	if err != nil {
		log.Printf("Error rolling back transaction: %s", err)
		return err
	}
	return cause
}

func (r *accessTokenRepositoryPostgres) FindByTokenRaw(accessToken string) (*AccessTokenRaw, error) {
	t := AccessTokenRaw{
		Token: &proto_oauth2.AccessToken{},
	}

	var parentToken sql.NullString

	err := util.QueryRow(r.statements, "find_by_token", accessToken).Scan(
		&t.Token.TokenType, &t.ClientId, &t.UserId, &t.Token.ExpiresIn,
		&t.Token.RefreshToken, &parentToken)

	if err != nil {
		return nil, err
	}
	t.Token.AccessToken = &accessToken
	if parentToken.Valid {
		t.ParentToken = &parentToken.String
	}
	return &t, nil
}

func (r *accessTokenRepositoryPostgres) FindUserForToken(
	accessToken *proto_oauth2.AccessToken) (*proto_user.User, error) {

	user := proto_user.User{}
	err := util.QueryRow(r.statements, "find_user_by_token", accessToken.GetAccessToken()).Scan(
		&user.Id, &user.DisplayName, &user.Email, &user.Password)

	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *accessTokenRepositoryPostgres) FindByRefreshToken(
	client *proto_oauth2.Client, refreshToken string) (*proto_oauth2.AccessToken, error) {

	at := proto_oauth2.AccessToken{}

	err := util.QueryRow(r.statements, "find_by_refresh_token", client.GetId(), refreshToken).Scan(
		&at.AccessToken, &at.TokenType, &at.ExpiresIn)

	if err != nil {
		return nil, err
	}
	at.RefreshToken = &refreshToken
	return &at, nil
}
