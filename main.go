package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"

	"github.com/opentarock/service-api/go/proto_oauth2"
	"github.com/opentarock/service-api/go/proto_user"
	"github.com/opentarock/service-user-management/nnservice"
	"github.com/opentarock/service-user-management/repository"
	"github.com/opentarock/service-user-management/service"
	"github.com/opentarock/service-user-management/util"
)

func main() {
	log.SetFlags(log.Ldate | log.Lmicroseconds)
	userService := nnservice.NewRepService("tcp://*:6001")
	oauth2Service := nnservice.NewRepService("tcp://*:6002")

	db, err := sql.Open("postgres", "user=postgres dbname=users sslmode=disable")
	if err != nil {
		log.Fatalf("Error connecting to database: %s", err)
	}
	defer db.Close()

	userRepository := repository.NewUserRepositoryPostgres(db)
	clientRepository := repository.NewClientRepositoryPostgres(db)
	accessTokenRepository := repository.NewAccessTokenRepositoryPostgres(db)

	tokenGenerator := util.NewRandTokenGenerator()

	userServiceHandlers := service.NewUserServiceHandlers(userRepository)
	userService.AddHandler(
		proto_user.RegisterUserMessage,
		userServiceHandlers.RegisterUserMessageHandler())
	userService.AddHandler(
		proto_user.AuthenticateUserMessage,
		userServiceHandlers.AuthenticateUserMessageHandler(tokenGenerator))
	go userService.Start()

	oauth2ServiceHandlers := service.NewOauth2ServiceHandlers(
		userRepository, clientRepository, accessTokenRepository)
	oauth2Service.AddHandler(
		proto_oauth2.AccessTokenAuthenticationMessage,
		oauth2ServiceHandlers.AccessTokenRequestHandler(tokenGenerator))
	oauth2Service.AddHandler(
		proto_oauth2.ValidateMessage,
		oauth2ServiceHandlers.ValidateHandler())
	oauth2Service.Start()
}
