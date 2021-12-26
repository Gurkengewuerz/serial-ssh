#!/bin/bash
USERNAME="gurken2108"
PROJECT="serial-ssh"
REGISTRY="docker.io"

docker buildx create --use --name ${USERNAME}-${PROJECT}
docker buildx build --platform linux/amd64,linux/arm/v7,linux/arm64 --push -t ${REGISTRY}/${USERNAME}/${PROJECT}:latest .
docker buildx stop ${USERNAME}-${PROJECT}
docker buildx rm ${USERNAME}-${PROJECT}

echo -e "Done!"

