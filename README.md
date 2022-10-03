# golang-notificator

Сервис для пересылки событий в мессенджеры

## Примеры отправки сообщений

```shell
curl --location --request POST 'localhost:10001' \
--form 'address="b6c4bca0b"' \
--form 'subject="Проверка"' \
--form 'text="Тест отправки сообщения в телеграм, слак и на почту одновременно"'
```

## Установка

### Добавления deploy ключа в git

```shell
cd ~/.ssh/
ssh-keygen
```
Имя файла указать golang-notificator (первый вопрос в ssh-keygen)

```shell
nano config
```

Добавить в файл config

```
Host golang-notificator.github.com
    HostName github.com
    IdentityFile ~/.ssh/golang-notificator
```

Добавить ключ в git

```shell
cat ~/.ssh/golang-notificator.pub
```

### Клонирование

```shell
cd /srv
sudo mkdir arhone
sudo chown $USER:$USER arhone
```
```shell
cd /srv/arhone
sudo mkdir golang-notificator
sudo chown $USER:$USER golang-notificator
```

```shell
git clone git@golang-notificator.github.com:arhone/golang-notificator.git ./golang-notificator
```
```shell
cd /srv/arhone/golang-notificator
```

## Настройка

```shell
cp config/main/config.example.json config/main/config.json
nano config/main/config.json
```

## Docker

### Создать .env из примера .env.example

```shell
cp .example.env .env
nano .env
```

### Сборка и запуск контейнера

```shell
sudo docker-compose -f docker-compose.yml up -d --build --remove-orphans
```

### Войти в контейнер

```shell
sudo docker exec -it golang-notificator-01 /bin/sh
```

### Остановка/Запуск контейнера

```shell
sudo docker stop golang-notificator-01
sudo docker start golang-notificator-01
```

### Просмотр логов через docker

```shell
sudo docker logs --tail 50 --follow --timestamps golang-notificator-01
```

## Deploy

```shell
. deploy.sh username@example.com
```

### Разрешить команду docker-compose без sudo

```shell
sudo visudo
```

#### Добавить запись
```
username ALL=NOPASSWD: /usr/bin/docker-compose
```

### Дополнительно

Возможно понадобится добавить ip сервиса в белые списки.

Получить base64 для Authorization: Basic
```shell
echo -n 'user:password' | base64
```

Убить все контейнеры
```shell
sudo docker kill $(docker ps -q)
```
