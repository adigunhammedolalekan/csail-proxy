#!/bin/bash
docker build -t "proxy" .
docker tag proxy registry.csail.app/proxy
docker push registry.csail.app/proxy