package service_test

import (
	"database/sql"
	"errors"
	"testing"

	"code.google.com/p/goprotobuf/proto"

	"github.com/opentarock/service-api/go/proto_user"
	"github.com/opentarock/service-user-management/nnservice"
	"github.com/opentarock/service-user-management/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type UserRepositoryMock struct {
	mock.Mock
}

func NewUserRepositoryMock() *UserRepositoryMock {
	return &UserRepositoryMock{}
}

func (r *UserRepositoryMock) Save(user *proto_user.User) (uint64, error) {
	args := r.Mock.Called(user)
	return uint64(args.Int(0)), args.Error(1)
}

func (r *UserRepositoryMock) FindByEmail(email string) (*proto_user.User, error) {
	args := r.Mock.Called(email)
	user, _ := args.Get(0).(*proto_user.User)
	return user, args.Error(1)
}

func (r *UserRepositoryMock) FindByEmailAndPassword(
	emailAddress, passwordPlain string) (*proto_user.User, error) {

	args := r.Mock.Called(emailAddress, passwordPlain)
	user, _ := args.Get(0).(*proto_user.User)
	return user, args.Error(1)
}

func (r *UserRepositoryMock) Count() (uint64, error) {
	args := r.Mock.Called()
	return uint64(args.Int(0)), args.Error(1)
}

type TokenGeneratorMock struct {
	mock.Mock
}

func NewTokenGeneratorMock() *TokenGeneratorMock {
	return &TokenGeneratorMock{}
}

func (m *TokenGeneratorMock) Generate(n uint) ([]byte, error) {
	args := m.Mock.Called(n)
	token, _ := args.Get(0).([]byte)
	return token, args.Error(1)
}

func handleMessage(t *testing.T, message proto.Message, handler nnservice.MessageHandler) []byte {
	messageData, err := proto.Marshal(message)
	assert.Nil(t, err)
	result := handler.HandleMessage(messageData)
	return result
}

func NewValidUser() *proto_user.User {
	return &proto_user.User{
		DisplayName: proto.String("name"),
		Email:       proto.String("mail@example.com"),
		Password:    proto.String("password"),
	}
}

func TestUserIsRegistered(t *testing.T) {
	userRepository := NewUserRepositoryMock()
	handlers := service.NewUserServiceHandlers(userRepository)

	registerUser := &proto_user.RegisterUser{
		User: NewValidUser(),
	}
	userRepository.On("Save", registerUser.GetUser()).Return(1, nil)
	result := handleMessage(t, registerUser, handlers.RegisterUserMessageHandler())
	var registerResponse proto_user.RegisterResponse
	err := proto.Unmarshal(result, &registerResponse)
	assert.Nil(t, err)
	assert.True(t, registerResponse.GetValid())
	assert.NotEmpty(t, registerResponse.GetRedirectUri())
	assert.NotEmpty(t, registerResponse.GetLocale())
}

func TestUserFieldDisplayNameIsValidated(t *testing.T) {
	handlers := service.NewUserServiceHandlers(nil)

	registerUser := &proto_user.RegisterUser{
		User: NewValidUser(),
	}
	registerUser.User.DisplayName = proto.String("ab")
	result := handleMessage(t, registerUser, handlers.RegisterUserMessageHandler())
	var registerResponse proto_user.RegisterResponse
	err := proto.Unmarshal(result, &registerResponse)
	assert.Nil(t, err)
	assert.False(t, registerResponse.GetValid())
	assert.Equal(t, 1, len(registerResponse.GetErrors()))
}

func TestUserFieldEmailIsValidated(t *testing.T) {
	handlers := service.NewUserServiceHandlers(nil)

	registerUser := &proto_user.RegisterUser{
		User: NewValidUser(),
	}
	registerUser.User.Email = proto.String("email")
	result := handleMessage(t, registerUser, handlers.RegisterUserMessageHandler())
	var registerResponse proto_user.RegisterResponse
	err := proto.Unmarshal(result, &registerResponse)
	assert.Nil(t, err)
	assert.False(t, registerResponse.GetValid())
	assert.Equal(t, 1, len(registerResponse.GetErrors()))
}

func TestUserFieldPasswordIsValidated(t *testing.T) {
	handlers := service.NewUserServiceHandlers(nil)

	registerUser := &proto_user.RegisterUser{
		User: NewValidUser(),
	}
	registerUser.User.Password = proto.String("pass")
	result := handleMessage(t, registerUser, handlers.RegisterUserMessageHandler())
	var registerResponse proto_user.RegisterResponse
	err := proto.Unmarshal(result, &registerResponse)
	assert.Nil(t, err)
	assert.False(t, registerResponse.GetValid())
	assert.Equal(t, 1, len(registerResponse.GetErrors()))
}

func TestUserAllFieldsAreValidatedAtOnce(t *testing.T) {
	handlers := service.NewUserServiceHandlers(nil)

	registerUser := &proto_user.RegisterUser{
		User: NewValidUser(),
	}
	registerUser.User.Email = proto.String("email")
	registerUser.User.Password = proto.String("pass")
	result := handleMessage(t, registerUser, handlers.RegisterUserMessageHandler())
	var registerResponse proto_user.RegisterResponse
	err := proto.Unmarshal(result, &registerResponse)
	assert.Nil(t, err)
	assert.False(t, registerResponse.GetValid())
	assert.Equal(t, 2, len(registerResponse.GetErrors()))
}

func TestTheUserIsAuthenticated(t *testing.T) {
	userRepository := NewUserRepositoryMock()
	tokenGenerator := NewTokenGeneratorMock()
	handlers := service.NewUserServiceHandlers(userRepository)

	user := NewValidUser()
	authUser := &proto_user.AuthenticateUser{
		Email:    user.Email,
		Password: user.Password,
	}
	userRepository.On("FindByEmailAndPassword", user.GetEmail(), user.GetPassword()).Return(user, nil)
	tokenGenerator.On("Generate", uint(64)).Return([]byte("session"), nil)

	result := handleMessage(t, authUser, handlers.AuthenticateUserMessageHandler(tokenGenerator))
	var authResult proto_user.AuthenticateResult
	err := proto.Unmarshal(result, &authResult)
	assert.Nil(t, err)
	assert.Equal(t, "73657373696f6e", authResult.GetSid())
}

func TestTheUnknownUserIsNotAuthenticated(t *testing.T) {
	userRepository := NewUserRepositoryMock()
	handlers := service.NewUserServiceHandlers(userRepository)

	user := NewValidUser()
	authUser := &proto_user.AuthenticateUser{
		Email:    user.Email,
		Password: user.Password,
	}
	userRepository.On("FindByEmailAndPassword",
		user.GetEmail(),
		user.GetPassword()).Return(nil, sql.ErrNoRows)

	result := handleMessage(t, authUser, handlers.AuthenticateUserMessageHandler(nil))
	var authResult proto_user.AuthenticateResult
	err := proto.Unmarshal(result, &authResult)
	assert.Nil(t, err)
	assert.Empty(t, authResult.GetSid())
}

func TestUserWithWrongPasswordIsNotAuthenticated(t *testing.T) {
	userRepository := NewUserRepositoryMock()
	handlers := service.NewUserServiceHandlers(userRepository)

	user := NewValidUser()
	authUser := &proto_user.AuthenticateUser{
		Email:    user.Email,
		Password: user.Password,
	}
	userRepository.On("FindByEmailAndPassword",
		user.GetEmail(),
		user.GetPassword()).Return(nil, errors.New("credentials_mismatch"))

	result := handleMessage(t, authUser, handlers.AuthenticateUserMessageHandler(nil))
	var authResult proto_user.AuthenticateResult
	err := proto.Unmarshal(result, &authResult)
	assert.Nil(t, err)
	assert.Empty(t, authResult.GetSid())
}
