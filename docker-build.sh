#!/bin/bash
USERNAME="gurken2108"
PROJECT="serial-ssh"
REGISTRY="docker.io"

docker buildx create --use --name ${USERNAME}-${PROJECT}
docker buildx build --platform linux/amd64,linux/arm/v7,linux/arm64 -t ${USERNAME}/${PROJECT}:latest .
docker tag ${USERNAME}/${PROJECT}:latest ${REGISTRY}/${USERNAME}/${PROJECT}:latest
docker push ${REGISTRY}/${USERNAME}/${PROJECT}

echo -e "Done!"

