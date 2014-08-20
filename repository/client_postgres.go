package repository

import (
	"database/sql"

	"github.com/opentarock/service-api/go/proto_oauth2"
	"github.com/opentarock/service-api/go/proto_user"
	"github.com/opentarock/service-user-management/util"
)

type clientRepositoryPostgres struct {
	db         *sql.DB
	statements map[string]*sql.Stmt
}

func NewClientRepositoryPostgres(db *sql.DB) *clientRepositoryPostgres {
	repo := &clientRepositoryPostgres{
		db:         db,
		statements: make(map[string]*sql.Stmt),
	}
	util.Prepare(db, repo.statements, "save_client",
		`INSERT INTO clients (client_id, client_secret, user_id)
		 VALUES ($1, $2, $3)`)
	util.Prepare(db, repo.statements, "find_client_by_id",
		`SELECT client_id, client_secret, user_id
		 FROM clients
		 WHERE client_id = $1`)
	return repo
}

func (r *clientRepositoryPostgres) Save(user *proto_user.User, client *proto_oauth2.Client) error {
	_, err := util.Exec(r.statements, "save_client", client.GetId(), client.GetSecret(), user.GetId())
	return err
}

func (r *clientRepositoryPostgres) FindById(id string) (*proto_oauth2.Client, error) {
	var clientId, clientSecret string
	var userId uint64
	err := util.QueryRow(r.statements, "find_client_by_id", id).Scan(
		&clientId, &clientSecret, &userId)
	if err != nil {
		return nil, err
	}
	client := proto_oauth2.Client{
		Id:     &clientId,
		Secret: &clientSecret,
	}
	return &client, nil
}
