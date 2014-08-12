package repository

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"

	"crypto/subtle"

	"code.google.com/p/goprotobuf/proto"

	"github.com/opentarock/service-api/go/proto_user"
	"github.com/opentarock/service-user-management/util"
	"github.com/opentarock/service-user-management/util/logutil"
)

const saltLength = 60

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
	repo.prepare("save_user", `INSERT INTO users (display_name, email, password, salt)
									  VALUES ($1, $2, $3, $4)
									  RETURNING id`)
	repo.prepare("find_user_by_email", `SELECT id, display_name, email, password, salt
										FROM users
										WHERE email = $1`)
	repo.prepare("count", `SELECT COUNT(*)
						   FROM users`)

	repo.Hasher = util.NewPBKDF2PasswordHasher()
	repo.TokenGenerator = util.NewRandTokenGenerator()
	return repo
}

func (r *userRepositoryPostgres) prepare(name, query string) error {
	if _, ok := r.statements[name]; !ok {
		stmt, err := r.db.Prepare(query)
		if err != nil {
			logutil.ErrorFatal(fmt.Sprintf("Error preparing statement %s", name), err)
		}
		r.statements[name] = stmt
		return nil
	}
	panic(fmt.Sprintf("Statement %s already exists", name))
}

func (r *userRepositoryPostgres) Save(user *proto_user.User) (uint64, error) {
	tokenRaw, err := r.TokenGenerator.Generate(saltLength)
	if err != nil {
		return 0, err
	}
	token := hex.EncodeToString(tokenRaw)
	passwordHash := r.hashPassword(user.GetPassword(), token)
	var id uint64
	err = r.queryRow("save_user", user.GetDisplayName(), user.GetEmail(), passwordHash, token).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *userRepositoryPostgres) FindByEmail(emailAddress string) (*proto_user.User, error) {
	userRaw, err := r.findByEmailRaw(emailAddress)
	if err != nil {
		return nil, err
	}
	return userRaw.User, nil
}

func (r *userRepositoryPostgres) findByEmailRaw(emailAddress string) (*UserRaw, error) {
	var id uint64
	var displayName, email, password, salt string
	err := r.queryRow("find_user_by_email", emailAddress).Scan(
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
	userRaw, err := r.findByEmailRaw(emailAddress)
	if err != nil {
		return nil, err
	}
	password := r.hashPassword(passwordPlain, userRaw.Salt)
	if subtle.ConstantTimeCompare([]byte(password), []byte(userRaw.User.GetPassword())) == 1 {
		return userRaw.User, nil
	}
	return nil, errors.New("credentials_mismatch")

}

func (r *userRepositoryPostgres) hashPassword(password, salt string) string {
	passwordHashRaw := r.Hasher.Hash(password, salt)
	return hex.EncodeToString(passwordHashRaw)
}

func (r *userRepositoryPostgres) Count() (uint64, error) {
	var count uint64
	err := r.queryRow("count").Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *userRepositoryPostgres) queryRow(name string, args ...interface{}) *sql.Row {
	if stmt, ok := r.statements[name]; ok {
		return stmt.QueryRow(args...)
	}
	panic(fmt.Sprintf("Statement not found: %s", name))
}

func (r *userRepositoryPostgres) Close() {
	for name, stmt := range r.statements {
		err := stmt.Close()
		if err != nil {
			log.Printf("Error closing statement '%s': %s", name, err)
		}
	}
}
