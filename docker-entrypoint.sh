#!/bin/bash

config='/srv/golang-notificator/config/main/config.json';
if [ ! -f $config ]; then
    cp /srv/golang-notificator/config/main/config.example.json $config;
fi

sed -i "s/\"debug\": false/\"debug\": $DEBUG/g" $config;

cat $config >> /srv/golang-notificator/logs/main.log

/srv/golang-notificator/main;
