package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"

	"github.com/opentarock/service-api/go/proto_user"
	"github.com/opentarock/service-user-management/nnservice"
	"github.com/opentarock/service-user-management/repository"
	"github.com/opentarock/service-user-management/service"
	"github.com/opentarock/service-user-management/util"
)

func main() {
	log.SetFlags(log.Ldate | log.Lmicroseconds)
	repService := nnservice.NewRepService("tcp://*:6001")

	db, err := sql.Open("postgres", "user=postgres dbname=users sslmode=disable")
	if err != nil {
		log.Fatalf("Error connecting to database: %s", err)
	}
	defer db.Close()

	userRepository := repository.NewUserRepositoryPostgres(db)

	tokenGenerator := util.NewRandTokenGenerator()

	userServiceHandlers := service.NewUserServiceHandlers(userRepository)
	repService.AddHandler(
		proto_user.RegisterUserMessage,
		userServiceHandlers.RegisterUserMessageHandler())
	repService.AddHandler(
		proto_user.AuthenticateUserMessage,
		userServiceHandlers.AuthenticateUserMessageHandler(tokenGenerator))
	repService.Start()
}
