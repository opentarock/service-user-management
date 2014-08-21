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
		if err == sql.ErrNoRows ||
			len(client.GetSecret()) != len(accessTokenRequest.GetClient().GetSecret()) ||
			subtle.ConstantTimeCompare([]byte(client.GetSecret()), []byte(accessTokenRequest.GetClient().GetSecret())) != 1 {

			accessTokenResponse.Error = &proto_oauth2.ErrorResponse{
				Error:            proto.String(oauth2.ErrorInvalidClient),
				ErrorDescription: proto.String("Client not found."),
			}
			log.Printf("Unknown client: %s", accessTokenRequest.GetClient().GetId())
		} else if err != nil {
			logutil.ErrorNormal("Error retrieving client", err)
			return nil
		} else {
			if request.GetGrantType() != oauth2.GrantTypePassword {
				accessTokenResponse = &proto_oauth2.AccessTokenResponse{
					Error: &proto_oauth2.ErrorResponse{
						Error:            proto.String(oauth2.ErrorUnsupportedGrantType),
						ErrorDescription: proto.String(fmt.Sprintf("Unsupported grant type: %s.", request.GetGrantType())),
					},
				}
			} else {
				user, err := s.userRepository.FindByEmailAndPassword(request.GetUsername(), request.GetPassword())
				if err != nil {
					if err == repository.ErrCredentialsMismatch {
						accessTokenResponse = &proto_oauth2.AccessTokenResponse{
							Error: &proto_oauth2.ErrorResponse{
								Error:            proto.String(oauth2.ErrorInvalidRequest),
								ErrorDescription: proto.String("Wrong owner credentials"),
							},
						}
					} else {
						logutil.ErrorNormal("Error retrieving user", err)
						return nil
					}
				} else {
					token, errToken := tokenGenerator.GenerateHex(accessTokenSize)
					refreshToken, errRefreshToken := tokenGenerator.GenerateHex(refreshTokenSize)
					if errToken != nil || errRefreshToken != nil {
						logutil.ErrorNormal("Error generating token", err)
						return nil
					}
					accessTokenResponse.Token = &proto_oauth2.AccessToken{
						AccessToken:  &token,
						TokenType:    proto.String("Bearer"),
						ExpiresIn:    proto.Uint64(3600),
						RefreshToken: &refreshToken,
					}
					err = s.accessTokenRepository.Save(user, client, accessTokenResponse.Token)
					if err != nil {
						logutil.ErrorNormal("Error persisting token", err)
						return nil
					}
					log.Printf("Authenticated client: %s", client.GetId())
				}
			}
		}
		// response is successful only if error was not set
		accessTokenResponse.Success = proto.Bool(accessTokenResponse.Error == nil)
		responseData, err := proto.Marshal(accessTokenResponse)
		logutil.ErrorFatal("Error marshalling AccessTokenResponse", err)
		return responseData
	})
}
