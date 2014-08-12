package repository

import (
	"database/sql"
	"os/exec"
	"testing"

	"code.google.com/p/goprotobuf/proto"
	_ "github.com/lib/pq"

	"github.com/opentarock/service-api/go/proto_user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type UserRepositoryTestSuite struct {
	suite.Suite
	db         *sql.DB
	repository *userRepositoryPostgres
}

func (s *UserRepositoryTestSuite) SetupTest() {
	err := exec.Command("goose", "-env=test", "-path=../db", "up").Run()
	if err != nil {
		panic(err)
	}
	db, err := sql.Open("postgres", "user=postgres dbname=test sslmode=disable")
	assert.Nil(s.T(), err)
	s.db = db
	s.repository = NewUserRepositoryPostgres(db)
}

func (s *UserRepositoryTestSuite) TearDownTest() {
	s.repository.Close()
	s.db.Close()
	err := exec.Command("goose", "-env=test", "-path=../db", "down").Run()
	if err != nil {
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

func (s *UserRepositoryTestSuite) TestUserIsSaved() {
	user := NewUser()
	countBefore, err := s.repository.Count()
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 0, countBefore)
	id, err := s.repository.Save(user)
	assert.Nil(s.T(), err)
	assert.True(s.T(), id > 0)
	userRetrieved, err := s.repository.FindByEmail("email@example.com")
	assert.Equal(s.T(), "email@example.com", userRetrieved.GetEmail())
	assert.Equal(s.T(), "name", userRetrieved.GetDisplayName())
	assert.Equal(s.T(), 128, len(userRetrieved.GetPassword()))
}

func (s *UserRepositoryTestSuite) TestUserWithCorrectPasswordIsFound() {
	user := NewUser()
	_, err := s.repository.Save(user)
	assert.Nil(s.T(), err)
	userRetrieved, err := s.repository.FindByEmailAndPassword("email@example.com", "password")
	assert.Equal(s.T(), "email@example.com", userRetrieved.GetEmail())
}

func (s *UserRepositoryTestSuite) TestUserWithWrongPasswordIsNotFound() {
	user := NewUser()
	_, err := s.repository.Save(user)
	assert.Nil(s.T(), err)
	_, err = s.repository.FindByEmailAndPassword("email@example.com", "wrong")
	assert.NotNil(s.T(), err)
}

func TestUserRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(UserRepositoryTestSuite))
}
