package main

import (
	"github.com/devstackq/ForumX/config"
	_ "github.com/mattn/go-sqlite3"
)

// func init() {
// 	config.Init()
// }

func main() {
	config.Init()
}

// write correct email and password - clien side
//show error 404 page
//Обнаруживает ли проект неправильный адрес электронной почты или пароль? - password min 8 symbols, and lowerBig number pwd
//if cookie = 0, notify message  user, logout etc
//Представлен ли в проекте скрипт для создания образов и контейнеров? (используя скрипт для упрощения сборки)
//like, dislike - refactor
//обработать ошикбки, log
//errors check http etc

//name func, variable - norm
//slice - controller/model/view
//conf file
//check contreoller -> middleware -> check data from CLient todo

//redirect logout not work,  &create post

//checkCookieLife(now, w, r, cookie) - del cookie client and backend - redirect main page

//photo not required || set defauklt photo
//refactor function  -> Single responsibility, DRY

//design refactor
// pagination for posts
//google acc signin -> -> back signin ? what??

//start Auth
//google token, client id, event signin Google, -> get data User,
//Name. email, photo, -> then save Db. -> authorized Forum
// Logout event, logout system, delete cookie, logout Google
//272819090705-qu6arlmkvs66hc5fuvalv6liuf2n9fj8.apps.googleusercontent.com   || W42c6sfYqhPc4O5wXMobY3av
