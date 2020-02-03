#!/bin/bash

docker build -t "proxy" .
docker tag proxy registry.hostgolang.com/proxy
docker push registry.hostgolang.com/proxy