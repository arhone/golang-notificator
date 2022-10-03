#!/bin/bash

if [ -f .env ]
then
    export $(cat .env | xargs)
fi
deploy_ssh='DEPLOY_SSH_'${1^^}
ssh=${!deploy_ssh}

if [ "$ssh" ]; then

    ssh "$ssh" 'cd /srv/arhone/golang-notificator && git pull && sudo docker-compose -f docker-compose.yml up -d --build --remove-orphans'

    deployUser=$(git config --get user.deploy);
    if [ "$deployUser" == "" ]; then
        deployUser=$(git config --get user.name);
    fi
    deployMessage=$(git log -1 --pretty=%B);
    commitName=$(git log -1 --pretty=%H);
    branch=$(git rev-parse --abbrev-ref HEAD);

    status=$(ssh "$ssh" sudo docker ps --format '{{.Status}}' --filter name=golang-notificator-01);
    if [ "$status" == "" ]; then
        status="failed"
    fi

    echo $status

else
    echo "Укажите сервер: '. deploy.sh develop'"
fi

