ARG GOLANG_VERSION

FROM golang:${GOLANG_VERSION}-alpine

ARG TZ

RUN apk update \
    && apk upgrade \
    && apk add --no-cache bash \
    && apk add --no-cache tzdata \
    && cp /usr/share/zoneinfo/${TZ} /etc/localtime \
    && echo ${TZ} > /etc/timezone

RUN apk add build-base
RUN apk add git

WORKDIR /srv/golang-notificator

COPY ./config ./config
COPY ./template ./template
COPY ./main.go .
COPY ./.env .

RUN go mod init notificator
RUN go mod tidy
RUN go build main.go

COPY ./docker-entrypoint.sh .
RUN chmod 755 /srv/golang-notificator/docker-entrypoint.sh
ENTRYPOINT ["/srv/golang-notificator/docker-entrypoint.sh"]
