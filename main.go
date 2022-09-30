package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	log "github.com/sirupsen/logrus"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type (

	// AddressConfigTelegram - Настройки для отправки сообщения в телеграм
	AddressConfigTelegram struct {
		Message string `json:"message"`
		Token   string `json:"token"`
		Chats   []int  `json:"chats"`
	}

	// AddressConfigSlack - Настройки для отправки сообщения в slack
	AddressConfigSlack struct {
		Message string   `json:"message"`
		Token   string   `json:"token"`
		Users   []string `json:"users"`
	}

	// AddressConfigEmail - Настройки для отправки почты
	AddressConfigEmail struct {
		Sender  string   `json:"sender"`
		Subject string   `json:"subject"`
		Body    string   `json:"body"`
		Emails  []string `json:"emails"`
		SMTP    struct {
			User     string `json:"user"`
			Password string `json:"password"`
			Server   string `json:"server"`
			Port     int    `json:"port"`
			From     string `json:"from"`
		} `json:"smtp"`
	}

	// AddressConfig - Настройки получателя
	AddressConfig struct {
		Telegram AddressConfigTelegram `json:"telegram"`
		Slack    AddressConfigSlack    `json:"slack"`
		Email    AddressConfigEmail    `json:"email"`
	}

	// Address - Получатель сообщения
	Address struct {
		Handlers []string      `json:"handlers"`
		Config   AddressConfig `json:"config"`
	}

	// Config - Настройки сервиса
	Config struct {
		Port      int                `json:"port"`
		Debug     bool               `json:"debug"`
		Addresses map[string]Address `json:"addresses"`
	}

	// ResponseBody - Структура тела ответа
	ResponseBody struct {
		Status struct {
			Code    int    `json:"code"`
			Message string `json:"message,omitempty"`
		} `json:"status"`
		Meta struct {
			Handlers []string `json:"handlers"`
		} `json:"meta"`
	}

	// SMTPLoginAuth - Структура авторизации
	SMTPLoginAuth struct {
		username, password string
	}

	// TplVars - Интерфейс переменных шаблона
	TplVars interface{}

	// MailTplVars - Переменная главной страницы
	MailTplVars struct {
		Title string
		Body  template.HTML
	}

	// slackApiChatPostMessage - Отправка сообщений
	slackApiChatPostMessage struct {
		Ok    bool   `json:"ok"`
		Error string `json:"error"`
	}
)

// Хранилище настроек
var configStorage = Config{}

// Хранилище кэша шаблонов
var templateCacheStorage = map[string]*template.Template{}

// main - Запуск сервиса
func main() {

	// Подключение логирования
	if err := os.MkdirAll("logs", os.FileMode(0755)); err != nil {
		panic(err)
	}
	logFile, err := os.OpenFile("logs/main.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	log.SetOutput(logFile)
	log.SetReportCaller(true)
	log.SetFormatter(&log.JSONFormatter{})
	log.Info("Запуск сервиса")

	// Загрузка настроек
	configFile, err := ioutil.ReadFile("config/main/config.json")
	if err != nil {
		log.Error(err.Error())
	}
	if err = json.Unmarshal(configFile, &configStorage); err != nil {
		log.Error(err.Error())
	}

	if configStorage.Debug {
		log.SetLevel(log.DebugLevel)
	}
	log.Info("debug mod: " + strconv.FormatBool(configStorage.Debug))

	// Отдача статики
	http.Handle("/favicon.ico", http.FileServer(http.Dir("public")))
	http.Handle("/robots.txt", http.FileServer(http.Dir("public")))

	// Отдача динамики
	http.HandleFunc("/", router)
	err = http.ListenAndServe(":"+strconv.Itoa(configStorage.Port), nil)
	if err != nil {
		log.Error(err)
		return
	}

}

// router - Маршрутизация
func router(response http.ResponseWriter, request *http.Request) {

	request.URL.Path = filepath.Clean(request.URL.Path)
	responseBody := new(ResponseBody)
	requestAddress := request.FormValue("address")

	if requestAddress == "" {
		responseBody.Status.Code = http.StatusForbidden
		responseBody.Status.Message = "Доступ запрещён:" + request.URL.Path
		sendResponse(response, responseBody)
		log.Debug(responseBody)
		return
	}

	if request.Method == "POST" {

		if address, ok := configStorage.Addresses[requestAddress]; ok {

			for _, handler := range address.Handlers {

				if handler == "telegram" {

					parcelTelegram := address.Config.Telegram
					requestMessage := request.FormValue("message")
					if requestMessage != "" {
						parcelTelegram.Message = requestMessage
					}
					go handlerTelegram(parcelTelegram)

				} else if handler == "slack" {

					parcelSlack := address.Config.Slack
					requestMessage := request.FormValue("message")
					if requestMessage != "" {
						parcelSlack.Message = requestMessage
					}
					go handlerSlack(parcelSlack)

				} else if handler == "email" {

					parcelEmail := address.Config.Email
					requestSender := request.FormValue("sender")
					if requestSender != "" {
						parcelEmail.Sender = requestSender
					}
					requestSubject := request.FormValue("subject")
					if requestSubject != "" {
						parcelEmail.Subject = requestSubject
					}
					requestBody := request.FormValue("body")
					if requestBody != "" {
						parcelEmail.Body = requestBody
					}
					go handlerEmail(parcelEmail)

				}

			}

			responseBody.Status.Code = http.StatusOK
			responseBody.Status.Message = request.Method + ": " + request.URL.Path
			responseBody.Meta.Handlers = address.Handlers
			sendResponse(response, responseBody)
			log.Debug(responseBody)
			return

		}

	}

	responseBody.Status.Code = http.StatusNotFound
	responseBody.Status.Message = request.Method + ": " + request.URL.Path
	sendResponse(response, responseBody)
	log.Debug(responseBody)
	return

}

// sendMessageToSlack - Отправляет сообщения в slack
func sendMessageToSlack(token string, userId string, message string) bool {

	message = strings.Replace(url.QueryEscape(message), "+", "%20", -1)

	client := http.Client{Timeout: 10 * time.Second}
	request, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage?channel="+userId+"&as_user=true&unfurl_links=true&text="+message, nil)
	if err != nil {
		log.Error(err)
		return false
	}

	request.Header = http.Header{
		"Content-Type":  []string{"application/x-www-form-urlencoded"},
		"Authorization": []string{"Bearer " + token},
	}

	response, err := client.Do(request)
	if err != nil {
		log.Error(err)
		return false
	}

	body, readErr := ioutil.ReadAll(response.Body)
	if readErr != nil {
		log.Error(readErr)
		return false
	}

	slackApiChatPostMessage := slackApiChatPostMessage{}
	err = json.Unmarshal(body, &slackApiChatPostMessage)
	if err != nil {
		log.Error(err)
		return false
	}
	if slackApiChatPostMessage.Ok == true {
		return true
	}

	log.Debug(response)
	return false

}

// handlerSlack - Отправляет сообщения в слак
func handlerSlack(parcel AddressConfigSlack) bool {

	for _, userId := range parcel.Users {
		go sendMessageToSlack(parcel.Token, userId, parcel.Message)
	}

	return true

}

// sendMessageToTelegram - Отправляет сообщения в телеграм
func sendMessageToTelegram(token string, chatId int, message string) bool {

	response, err := http.Get(
		"https://api.telegram.org/bot" + token +
			"/sendMessage?chat_id=" + strconv.Itoa(chatId) +
			"&parse_mode=html&disable_web_page_preview=true&text=" + url.QueryEscape(message))
	if err != nil {
		log.Error(err)
		return false
	}

	log.Debug(response)
	return true

}

// handlerTelegram - Отправляет в телеграм
func handlerTelegram(parcel AddressConfigTelegram) bool {

	for _, chatId := range parcel.Chats {
		go sendMessageToTelegram(parcel.Token, chatId, parcel.Message)
	}

	return true

}

// handlerEmail - Отправляет сообщение на почту
func handlerEmail(parcel AddressConfigEmail) bool {

	var mailTplVars TplVars = MailTplVars{
		Title: parcel.Subject,
		Body:  template.HTML(parcel.Body),
	}

	from := "From: " + parcel.Sender + " <" + parcel.SMTP.From + ">\r\n"
	to := "To: " + strings.Join(parcel.Emails, ";") + "\r\n"
	subject := "Subject: " + parcel.Subject + " " + time.Now().Format("2006-01-02 15:04:05") + " \r\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"
	message := []byte(from + to + subject + mime + getRenderTemplate("template/mail.html", mailTplVars) + "\r\n")
	auth := getSMTPLoginAuth(parcel.SMTP.User, parcel.SMTP.Password)
	err := smtp.SendMail(parcel.SMTP.Server+":"+strconv.Itoa(parcel.SMTP.Port), auth, parcel.SMTP.From, parcel.Emails, message)
	if err != nil {
		log.Error(err)
		return false
	}

	log.Debug(subject + from + to)
	return true

}

// sendResponse - Отвечает клиенту
func sendResponse(response http.ResponseWriter, responseBody *ResponseBody) bool {

	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)

	data, err := json.Marshal(responseBody)
	if err != nil {
		if _, err := response.Write(data); err != nil {
			return false
		}
	}

	if _, err := response.Write(data); err != nil {
		return false
	}

	return true

}

// getRenderTemplate - Возвращает рендер шаблона
func getRenderTemplate(path string, tplVars TplVars) string {

	cacheKey := getMd5Hash(path)
	var html *template.Template
	if cacheHtml, ok := templateCacheStorage[cacheKey]; ok {
		html = cacheHtml
	} else {
		newHtml, err := template.ParseFiles(path)
		html = newHtml
		if err != nil {
			log.Error(err)
		} else {
			templateCacheStorage[cacheKey] = newHtml
		}
	}

	var tpl bytes.Buffer
	if err := html.Execute(&tpl, tplVars); err != nil {
		log.Error()
	}

	return tpl.String()

}

// getSMTPLoginAuth - Авторизацию smtp
func getSMTPLoginAuth(username, password string) smtp.Auth {
	return &SMTPLoginAuth{username, password}
}

// Start - ?
func (a *SMTPLoginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

// Next - ?
func (a *SMTPLoginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("Unkown fromServer")
		}
	}
	return nil, nil
}

// getMd5Hash - Возвращает md5 хэш
func getMd5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}
