package repository

import (
	"database/sql"
	"fmt"
	"os/exec"
	"testing"

	"code.google.com/p/gogoprotobuf/proto"
	_ "github.com/lib/pq"

	"github.com/opentarock/service-api/go/proto_oauth2"
	"github.com/opentarock/service-api/go/proto_user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type PostgresRepositoryTestSuite struct {
	suite.Suite
	db                    *sql.DB
	userRepository        *userRepositoryPostgres
	clientRepository      *clientRepositoryPostgres
	accessTokenRepository *accessTokenRepositoryPostgres
}

func (s *PostgresRepositoryTestSuite) SetupTest() {
	out, err := exec.Command(`psql`, `-h`, `localhost`, `-U`, `postgres`, `-c`, `CREATE DATABASE test`).CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		panic(err)
	}
	err = exec.Command("goose", "-env=test", "-path=../db", "up").Run()
	if err != nil {
		panic(err)
	}
	db, err := sql.Open("postgres", "user=postgres dbname=test sslmode=disable")
	assert.Nil(s.T(), err)
	s.db = db
	s.userRepository = NewUserRepositoryPostgres(db)
	s.clientRepository = NewClientRepositoryPostgres(db)
	s.accessTokenRepository = NewAccessTokenRepositoryPostgres(db)
}

func (s *PostgresRepositoryTestSuite) TearDownTest() {
	s.userRepository.Close()
	s.db.Close()
	out, err := exec.Command(`psql`, `-h`, `localhost`, `-U`, `postgres`, `-c`, `DROP DATABASE IF EXISTS test`).CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		panic(err)
	}
}

func NewUser() *proto_user.User {
	return &proto_user.User{
		DisplayName: proto.String("name"),
		Email:       proto.String("email@example.com"),
		Password:    proto.String("password"),
	}
}

func NewClient() *proto_oauth2.Client {
	return &proto_oauth2.Client{
		Id:     proto.String("client_id"),
		Secret: proto.String("client_secret"),
	}
}

func NewAccessToken() *proto_oauth2.AccessToken {
	return &proto_oauth2.AccessToken{
		AccessToken:  proto.String("token"),
		TokenType:    proto.String("type"),
		ExpiresIn:    proto.Uint64(3600),
		RefreshToken: proto.String("refresh"),
	}

}

func (s *PostgresRepositoryTestSuite) TestUserIsSaved() {
	user := NewUser()
	countBefore, err := s.userRepository.Count()
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 0, countBefore)
	err = s.userRepository.Save(user)
	assert.Nil(s.T(), err)
	assert.True(s.T(), user.GetId() > 0)
	userRetrieved, err := s.userRepository.FindByEmail("email@example.com")
	assert.Equal(s.T(), "email@example.com", userRetrieved.GetEmail())
	assert.Equal(s.T(), "name", userRetrieved.GetDisplayName())
	assert.Equal(s.T(), 128, len(userRetrieved.GetPassword()))
}

func (s *PostgresRepositoryTestSuite) TestUserWithCorrectPasswordIsFound() {
	user := NewUser()
	err := s.userRepository.Save(user)
	assert.Nil(s.T(), err)
	userRetrieved, err := s.userRepository.FindByEmailAndPassword("email@example.com", "password")
	assert.Equal(s.T(), "email@example.com", userRetrieved.GetEmail())
}

func (s *PostgresRepositoryTestSuite) TestUserWithWrongPasswordIsNotFound() {
	user := NewUser()
	err := s.userRepository.Save(user)
	assert.Nil(s.T(), err)
	_, err = s.userRepository.FindByEmailAndPassword("email@example.com", "wrong")
	assert.NotNil(s.T(), err)
}

func (s *PostgresRepositoryTestSuite) TestClientIsSaved() {
	user := NewUser()
	s.userRepository.Save(user)
	client := NewClient()
	err := s.clientRepository.Save(user, client)
	assert.Nil(s.T(), err)
	clientRetrieved, err := s.clientRepository.FindById("client_id")
	assert.Equal(s.T(), "client_id", clientRetrieved.GetId())
	assert.Equal(s.T(), "client_secret", clientRetrieved.GetSecret())
}

func (s *PostgresRepositoryTestSuite) TestAccessTokenIsSaved() {
	user := NewUser()
	s.userRepository.Save(user)
	client := NewClient()
	s.clientRepository.Save(user, client)
	accessToken := NewAccessToken()
	err := s.accessTokenRepository.Save(user, client, accessToken, nil)
	assert.Nil(s.T(), err)
	accessTokenRetrieved, err := s.accessTokenRepository.FindByRefreshToken(client, accessToken.GetRefreshToken())
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), accessToken, accessTokenRetrieved)
}

func (s *PostgresRepositoryTestSuite) TestCanRetrieveUserForAccessToken() {
	user := NewUser()
	s.userRepository.Save(user)
	client := NewClient()
	s.clientRepository.Save(user, client)
	accessToken := NewAccessToken()
	err := s.accessTokenRepository.Save(user, client, accessToken, nil)
	assert.Nil(s.T(), err)
	retrievedUser, err := s.accessTokenRepository.FindUserForToken(accessToken)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), user.GetId(), retrievedUser.GetId())
	assert.Equal(s.T(), user.GetEmail(), retrievedUser.GetEmail())
	assert.Equal(s.T(), user.GetDisplayName(), retrievedUser.GetDisplayName())
	assert.NotEmpty(s.T(), user.GetPassword())
}

func TestPostgresRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresRepositoryTestSuite))
}
