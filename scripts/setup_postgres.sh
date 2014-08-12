#!/usr/bin/env bash

docker run --name=user-service-postgres -p 5432:5432 -d postgres:9.3
createdb -h localhost -U postgres users
