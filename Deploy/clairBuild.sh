#!/usr/bin/env bash

cp ./clair-config/docker-compose.yml $HOME/docker-compose.yml

mkdir $HOME/clair_config

cp ./clair-config/config.yaml $HOME/clair_config/config.yaml

docker-compose -f $HOME/docker-compose.yml up --build -d

docker start clair_clair

#go get -u github.com/coreos/clair/contrib/analyze-local-images