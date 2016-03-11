#!/bin/bash

docker build -t hydra.sky:5000/dockersurefire .
docker push hydra.sky:5000/dockersurefire
