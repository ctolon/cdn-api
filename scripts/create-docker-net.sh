#!/bin/bash

NETWORK_NAME="cdn_net"
SUBNET="10.20.0.0/16"
GATEWAY="10.20.0.1"

if docker network ls | grep -q "$NETWORK_NAME"; then
    echo "Network '$NETWORK_NAME' already exist."
else
    docker network create \
      --driver bridge \
      --subnet $SUBNET \
      --gateway $GATEWAY \
      $NETWORK_NAME

    if [ $? -eq 0 ]; then
        echo "Network '$NETWORK_NAME' created o.k.."
    else
        echo "Network '$NETWORK_NAME' failed on create!"
        exit 1
    fi
fi