package service

import (
	"database/sql"
	"fmt"
	"log"

	"crypto/subtle"

	"code.google.com/p/gogoprotobuf/proto"
	"github.com/arjantop/oauth2-util"
	"github.com/opentarock/service-api/go/proto_oauth2"
	"github.com/opentarock/service-user-management/nnservice"
	"github.com/opentarock/service-user-management/repository"
	"github.com/opentarock/service-user-management/util"
	"github.com/opentarock/service-user-management/util/logutil"
)

const (
	accessTokenSize  = 32
	refreshTokenSize = 32
)

type oauth2ServiceHandlers struct {
	userRepository        repository.UserRepository
	clientRepository      repository.ClientRepository
	accessTokenRepository repository.AccessTokenRepository
}

func NewOauth2ServiceHandlers(
	userRepository repository.UserRepository,
	clientRepository repository.ClientRepository,
	accessTokenRepository repository.AccessTokenRepository) *oauth2ServiceHandlers {

	return &oauth2ServiceHandlers{
		userRepository:        userRepository,
		clientRepository:      clientRepository,
		accessTokenRepository: accessTokenRepository,
	}
}

func (s *oauth2ServiceHandlers) AccessTokenRequestHandler(tokenGenerator util.TokenGenerator) nnservice.MessageHandler {
	return nnservice.MessageHandlerFunc(func(data []byte) []byte {
		accessTokenRequest := &proto_oauth2.AccessTokenAuthentication{}
		err := proto.Unmarshal(data, accessTokenRequest)
		if err != nil {
			logutil.ErrorNormal("Error unmarshalling AccessTokenRequest", err)
			return nil
		}
		accessTokenResponse := &proto_oauth2.AccessTokenResponse{}
		request := accessTokenRequest.GetRequest()
		client, err := s.clientRepository.FindById(accessTokenRequest.GetClient().GetId())
		if err == sql.ErrNoRows || !clientEquals(client, accessTokenRequest.GetClient()) {
			accessTokenResponse.Error = &proto_oauth2.ErrorResponse{
				Error:            proto.String(oauth2.ErrorInvalidClient),
				ErrorDescription: proto.String("Client not found."),
			}
			log.Printf("Unknown client: %s", accessTokenRequest.GetClient().GetId())
		} else if err != nil {
			logutil.ErrorNormal("Error retrieving client", err)
			return nil
		} else {
			var err error
			switch request.GetGrantType() {
			case oauth2.GrantTypePassword:
				accessTokenResponse, err = s.handleGrantTypePassword(tokenGenerator, client, request)
			case oauth2.GrantTypeRefreshToken:
				accessTokenResponse, err = s.handleGrantTypeRefreshToken(tokenGenerator, client, request)
			default:
				accessTokenResponse = &proto_oauth2.AccessTokenResponse{
					Error: &proto_oauth2.ErrorResponse{
						Error:            proto.String(oauth2.ErrorUnsupportedGrantType),
						ErrorDescription: proto.String(fmt.Sprintf("Unsupported grant type: %s.", request.GetGrantType())),
					},
				}
			}
			if err != nil {
				log.Println(err)
				return nil
			}
		}
		// response is successful only if error was not set
		accessTokenResponse.Success = proto.Bool(accessTokenResponse.Error == nil)
		responseData, err := proto.Marshal(accessTokenResponse)
		logutil.ErrorFatal("Error marshalling AccessTokenResponse", err)
		return responseData
	})
}

func clientEquals(client, clientOther *proto_oauth2.Client) bool {
	return len(client.GetSecret()) == len(clientOther.GetSecret()) &&
		subtle.ConstantTimeCompare([]byte(client.GetSecret()), []byte(clientOther.GetSecret())) == 1
}

func (s *oauth2ServiceHandlers) handleGrantTypePassword(
	tokenGenerator util.TokenGenerator,
	client *proto_oauth2.Client,
	request *proto_oauth2.AccessTokenRequest) (*proto_oauth2.AccessTokenResponse, error) {

	accessTokenResponse := &proto_oauth2.AccessTokenResponse{}

	user, err := s.userRepository.FindByEmailAndPassword(request.GetUsername(), request.GetPassword())
	if err != nil {
		if err == repository.ErrCredentialsMismatch {
			accessTokenResponse = &proto_oauth2.AccessTokenResponse{
				Error: &proto_oauth2.ErrorResponse{
					Error:            proto.String(oauth2.ErrorInvalidGrant),
					ErrorDescription: proto.String("Wrong owner credentials"),
				},
			}
		} else {
			return nil, fmt.Errorf("Error retrieving user: %s", err)
		}
	} else {
		token, err := generateToken(tokenGenerator)
		if err != nil {
			return nil, fmt.Errorf("Error generating new token: %s", err)
		}
		accessTokenResponse.Token = token
		err = s.accessTokenRepository.Save(user, client, accessTokenResponse.Token, nil)
		if err != nil {
			return nil, fmt.Errorf("Error persisting token: %s", err)
		}
		log.Printf("Authenticated client: %s", client.GetId())
	}
	return accessTokenResponse, nil
}

func (s *oauth2ServiceHandlers) handleGrantTypeRefreshToken(
	tokenGenerator util.TokenGenerator,
	client *proto_oauth2.Client,
	request *proto_oauth2.AccessTokenRequest) (*proto_oauth2.AccessTokenResponse, error) {

	accessTokenResponse := &proto_oauth2.AccessTokenResponse{}

	if request.GetRefreshToken() == "" {
		accessTokenResponse.Error = &proto_oauth2.ErrorResponse{
			Error:            proto.String(oauth2.ErrorInvalidRequest),
			ErrorDescription: proto.String(fmt.Sprintf("Required paremeter is missing: %s", oauth2.ParameterRefreshToken)),
		}
		return accessTokenResponse, nil
	}

	currentToken, err := s.accessTokenRepository.FindByRefreshToken(client, request.GetRefreshToken())
	if err == sql.ErrNoRows {
		accessTokenResponse.Error = &proto_oauth2.ErrorResponse{
			Error:            proto.String(oauth2.ErrorInvalidGrant),
			ErrorDescription: proto.String("Invalid refresh token"),
		}
		log.Printf("Refresh token %s not found", request.GetRefreshToken())
		return accessTokenResponse, nil
	} else if err != nil {
		return nil, fmt.Errorf("Error retrieving token: %s", err)
	}

	newToken, err := generateToken(tokenGenerator)
	if err != nil {
		return nil, fmt.Errorf("Error generating refreshed token: %s", err)
	}
	user, err := s.accessTokenRepository.FindUserForToken(currentToken)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving user: %s", err)
	}
	err = s.accessTokenRepository.Save(user, client, newToken, currentToken)
	if err != nil {
		return nil, fmt.Errorf("Error persisting token", err)
	}

	accessTokenResponse.Token = newToken

	return accessTokenResponse, nil
}

func generateToken(tokenGenerator util.TokenGenerator) (*proto_oauth2.AccessToken, error) {
	token, err := tokenGenerator.GenerateHex(accessTokenSize)
	if err != nil {
		return nil, err
	}
	refreshToken, err := tokenGenerator.GenerateHex(refreshTokenSize)
	if err != nil {
		return nil, err
	}
	return &proto_oauth2.AccessToken{
		AccessToken:  &token,
		TokenType:    proto.String("Bearer"),
		ExpiresIn:    proto.Uint64(12 * 3600), // expires in 12 hours
		RefreshToken: &refreshToken,
	}, nil
}
