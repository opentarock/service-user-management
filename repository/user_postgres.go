package repository

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"log"

	"crypto/subtle"

	"code.google.com/p/gogoprotobuf/proto"

	"github.com/opentarock/service-api/go/proto_user"
	"github.com/opentarock/service-user-management/util"
)

const (
	saltLength = 60
)

var ErrCredentialsMismatch = errors.New("userRepository: credentials_mismatch")

type userRepositoryPostgres struct {
	db             *sql.DB
	statements     map[string]*sql.Stmt
	Hasher         util.PasswordHasher
	TokenGenerator util.TokenGenerator
}

func NewUserRepositoryPostgres(db *sql.DB) *userRepositoryPostgres {
	repo := &userRepositoryPostgres{
		db:         db,
		statements: make(map[string]*sql.Stmt),
	}
	util.Prepare(db, repo.statements, "save_user",
		`INSERT INTO users (display_name, email, password, salt)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`)
	util.Prepare(db, repo.statements, "find_user_by_id",
		`SELECT id, display_name, email, password, salt
		 FROM users
		 WHERE id = $1`)
	util.Prepare(db, repo.statements, "find_user_by_email",
		`SELECT id, display_name, email, password, salt
		 FROM users
		 WHERE email = $1`)
	util.Prepare(db, repo.statements, "count",
		`SELECT COUNT(*)
		 FROM users`)

	repo.Hasher = util.NewPBKDF2PasswordHasher()
	repo.TokenGenerator = util.NewRandTokenGenerator()
	return repo
}

func (r *userRepositoryPostgres) Save(user *proto_user.User) error {
	token, err := r.TokenGenerator.GenerateHex(saltLength)
	if err != nil {
		return err
	}
	passwordHash := r.hashPassword(user.GetPassword(), token)
	var id uint64
	err = util.QueryRow(r.statements, "save_user", user.GetDisplayName(), user.GetEmail(), passwordHash, token).Scan(&id)
	if err != nil {
		return err
	}
	user.Id = proto.Uint64(id)
	return nil
}

func (r *userRepositoryPostgres) FindById(id uint64) (*proto_user.User, error) {
	userRaw, err := r.findRaw("find_user_by_id", id)
	if err != nil {
		return nil, err
	}
	return userRaw.User, nil
}

func (r *userRepositoryPostgres) FindByEmail(emailAddress string) (*proto_user.User, error) {
	userRaw, err := r.findRaw("find_user_by_email", emailAddress)
	if err != nil {
		return nil, err
	}
	return userRaw.User, nil
}

func (r *userRepositoryPostgres) findRaw(query string, args ...interface{}) (*UserRaw, error) {
	var id uint64
	var displayName, email, password, salt string
	err := util.QueryRow(r.statements, "find_user_by_email", args...).Scan(
		&id, &displayName, &email, &password, &salt)
	if err != nil {
		return nil, err
	}
	user := &proto_user.User{
		Id:          proto.Uint64(id),
		DisplayName: proto.String(displayName),
		Email:       proto.String(email),
		Password:    proto.String(password),
	}
	userRaw := UserRaw{
		User: user,
		Salt: salt,
	}
	return &userRaw, nil
}

func (r *userRepositoryPostgres) FindByEmailAndPassword(emailAddress, passwordPlain string) (*proto_user.User, error) {
	userRaw, err := r.findRaw("find_user_by_email", emailAddress)
	if err == sql.ErrNoRows {
		return nil, ErrCredentialsMismatch
	} else if err != nil {
		return nil, err
	}
	password := r.hashPassword(passwordPlain, userRaw.Salt)
	if subtle.ConstantTimeCompare([]byte(password), []byte(userRaw.User.GetPassword())) == 1 {
		return userRaw.User, nil
	}
	return nil, ErrCredentialsMismatch

}

func (r *userRepositoryPostgres) hashPassword(password, salt string) string {
	passwordHashRaw := r.Hasher.Hash(password, salt)
	return hex.EncodeToString(passwordHashRaw)
}

func (r *userRepositoryPostgres) Count() (uint64, error) {
	var count uint64
	err := util.QueryRow(r.statements, "count").Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *userRepositoryPostgres) Close() {
	for name, stmt := range r.statements {
		err := stmt.Close()
		if err != nil {
			log.Printf("Error closing statement '%s': %s", name, err)
		}
	}
}
