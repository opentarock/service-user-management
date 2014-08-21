package service

import (
	"database/sql"
	"log"
	"strings"
	"unicode/utf8"

	"code.google.com/p/gogoprotobuf/proto"

	"github.com/opentarock/service-api/go/proto_user"
	"github.com/opentarock/service-user-management/nnservice"
	"github.com/opentarock/service-user-management/repository"
	"github.com/opentarock/service-user-management/util"
	"github.com/opentarock/service-user-management/util/logutil"
)

const sessionIdLength = 64

type userServiceHandlers struct {
	userRepository repository.UserRepository
}

func NewUserServiceHandlers(userRepository repository.UserRepository) *userServiceHandlers {
	return &userServiceHandlers{
		userRepository: userRepository,
	}
}

func (s *userServiceHandlers) RegisterUserMessageHandler() nnservice.MessageHandler {
	return nnservice.MessageHandlerFunc(func(data []byte) []byte {
		registerUser := &proto_user.RegisterUser{}
		err := proto.Unmarshal(data, registerUser)
		if err != nil {
			logutil.ErrorNormal("Error unmarshalling RegisterUser", err)
			return nil
		}

		var registerResponse *proto_user.RegisterResponse
		if errors := s.validateUser(registerUser.GetLocale(), registerUser.GetUser()); len(errors) != 0 {
			registerResponse = &proto_user.RegisterResponse{
				Valid:  proto.Bool(false),
				Errors: errors,
			}
		} else {
			err := s.userRepository.Save(registerUser.GetUser())
			if err != nil {
				logutil.ErrorNormal("Error inserting user", err)
				return nil
			}
			log.Printf("Registered user: id=%d", registerUser.GetUser().GetId())

			registerResponse = &proto_user.RegisterResponse{
				Valid:       proto.Bool(true),
				RedirectUri: proto.String("http://localhost:8080/user"), // TODO: implement redirect uri checking
			}
		}
		registerResponse.Locale = proto.String("en") // TODO: implement i18n

		responseData, err := proto.Marshal(registerResponse)
		logutil.ErrorFatal("Error marshalling RegisterResponse", err)
		return responseData
	})
}

func (s *userServiceHandlers) validateUser(
	locale string, user *proto_user.User) []*proto_user.RegisterResponse_InputError {

	errors := make([]*proto_user.RegisterResponse_InputError, 0)
	displayNameError := s.validateDisplayName(user.GetDisplayName())
	if displayNameError != nil {
		errors = append(errors, displayNameError)
	}
	emailError := s.validateEmail(user.GetEmail())
	if emailError != nil {
		errors = append(errors, emailError)
	}
	passwordError := s.validatePassword(user.GetPassword())
	if passwordError != nil {
		errors = append(errors, passwordError)
	}
	return errors
}

func (s *userServiceHandlers) validateDisplayName(displayName string) *proto_user.RegisterResponse_InputError {
	displayName = strings.TrimSpace(displayName)
	var errorMessage string
	if displayName == "" {
		errorMessage = "Display Name must not be empty."
	} else if strlen(displayName) < 3 || strlen(displayName) > 20 {
		errorMessage = "Display Name length must be between 3 and 20 characters."
	}
	if errorMessage != "" {
		return proto_user.NewInputError("display_name", errorMessage)
	}
	return nil
}

func (s *userServiceHandlers) validateEmail(email string) *proto_user.RegisterResponse_InputError {
	email = strings.TrimSpace(email)
	var errorMessage string
	if email == "" {
		errorMessage = "Email must not be empty."
	} else if !strings.Contains(email, "@") {
		errorMessage = "Email must contain an at sign (@)."
	}
	if errorMessage != "" {
		return proto_user.NewInputError("email", errorMessage)
	}
	return nil
}

func (s *userServiceHandlers) validatePassword(password string) *proto_user.RegisterResponse_InputError {
	var errorMessage string
	if password == "" {
		errorMessage = "Password must not be empty."
	} else if strlen(password) < 6 {
		errorMessage = "Password must be at least 6 characters long."
	} else if strlen(password) > 1024 {
		errorMessage = "Password length must not exceed 1024 characters."
	}
	if errorMessage != "" {
		return proto_user.NewInputError("password", errorMessage)
	}
	return nil
}

func (s *userServiceHandlers) AuthenticateUserMessageHandler(tokenGenerator util.TokenGenerator) nnservice.MessageHandler {
	return nnservice.MessageHandlerFunc(func(data []byte) []byte {
		authUser := &proto_user.AuthenticateUser{}
		err := proto.Unmarshal(data, authUser)
		if err != nil {
			logutil.ErrorNormal("Error unmarshalling AuthenticateUser", err)
			return nil
		}

		authResult := &proto_user.AuthenticateResult{
			Locale: proto.String("en"), // TODO: implement i18n
		}

		user, err := s.userRepository.FindByEmailAndPassword(authUser.GetEmail(), authUser.GetPassword())
		// If there are no rows returned from the query user authentication automatically fails.
		if err != nil && err != sql.ErrNoRows {
			logutil.ErrorNormal("Error retrieving user with given password", err)
		} else if err == nil {
			log.Printf("Authenticated user id=%d", user.GetId())
			sessionId, err := tokenGenerator.GenerateHex(sessionIdLength)
			if err != nil {
				return nil
			}
			authResult.Sid = proto.String(sessionId)
		} else {
			log.Printf("User not found: email=%s", authUser.GetEmail())
		}

		responseData, err := proto.Marshal(authResult)
		logutil.ErrorFatal("Error marshalling AuthenticateResult", err)
		return responseData
	})
}

func strlen(str string) int {
	return utf8.RuneCountInString(str)
}
