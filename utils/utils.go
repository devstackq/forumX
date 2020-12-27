package utils

import (
	"ForumX/general"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	//DB connect
	DB   *sql.DB
	err  error
	temp = template.Must(template.ParseFiles("./view/header.html", "view/update_comment.html", "view/activity.html", "view/disliked.html", "view/category_post.html", "view/favorites.html", "view/404page.html", "view/update_post.html", "view/created_post.html", "view/comment_user.html", "view/profile_update.html", "view/search.html", "view/another_user.html", "view/profile.html", "view/signin.html", "view/signup.html", "view/filter.html", "view/post.html", "view/comment_post.html", "view/create_post.html", "view/footer.html", "view/index.html"))

	GoogleConfig = &oauth2.Config{
		RedirectURL:  "http://localhost:6969/googleUserInfo",
		ClientID:     "154015070566-3s9nqt7qoe3dlhopeje85buq89603hae",
		ClientSecret: "HtjxrjYxw8g4WmvzQvsv9Efu",
		Scopes: []string{"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint: google.Endpoint,
	}
	Code     string
	Token    string
	AuthType string
)
//API struct
type API struct {
	Authenticated bool `json:"authenticated"`
}

//IsAuth check user now authorized system ?
func IsAuth(r *http.Request) API {
	var auth API
	for _, cookie := range r.Cookies() {
		if cookie.Name == "_cookie" {
			auth.Authenticated = true
		}
	}
	return auth
}

//IsCookie check cookie all case
func IsCookie(w http.ResponseWriter, r *http.Request, cookie string) (bool, general.Session) {

	s := general.Session{UUID: cookie}
	//if have _cookie in  browser, get userId - in  session table ->
	if IsAuth(r).Authenticated {
		// get userid by borswerCookie
		err = DB.QueryRow("SELECT user_id FROM session WHERE uuid = ?", s.UUID).Scan(&s.UserID)
		if err != nil {
			log.Println(err, "warn: no session with this cookie")
			Logout(w, r, &s)
			return false, s
		}
		//get uuid by userId in session table
		err = DB.QueryRow("SELECT uuid FROM session WHERE user_id = ?", s.UserID).Scan(&s.DbCookie)
		if err != nil {
			log.Println(err, "warn: no session with this UserID")
			Logout(w, r, &s)
			return false, s
		}
		//sesseionCookie != dbCookie
		if s.DbCookie != s.UUID {
			log.Println(err, "warn: reSession || cookie changed")
			Logout(w, r, &s)
			//s = general.Session{}
			return false, s
		}
	}
	return true, s
}

//CheckLetter correct letter
func CheckLetter(value string) bool {
	for _, v := range value {
		if v >= 97 && v <= 122 || v >= 65 && v <= 90 || v >= 32 && v <= 64 || v > 128 {
			return true
		}
	}
	return false
}

//calback methods continue, anonim
func TestCallback(arr []int, count int, flag bool, sortX func([]int)) {
	if count > 5 && flag {
		sortX(arr)
	} else {
		//1 call func example
		fmt.Println("не сооттветствуют данные")
	}
}
//CheckMethod anonim callback function, call parent Func -> then call child Func if condition True
func CheckMethod(method string, tmpl string, isAuth bool, msg string, w http.ResponseWriter, f func(http.ResponseWriter)) {
	 if (method == "GET" ) &&  (tmpl == "signin" || tmpl == "signup") {
		//RenderTemplate(w, "header", isAuth)
		RenderTemplate(w, tmpl, isAuth)
	} else {
		f(w)
	}
}

//utils.RenderTemplate function, 500 error, if ok render page
func RenderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err = temp.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		log.Println(err, "exec templ ERR")
		w.WriteHeader(500)
		fmt.Fprintf(w, "Internal server error, 500")
		return
	}
}

//IsCookieExpiration if cookie time = 0, delete session and cookie client
func Logout(w http.ResponseWriter, r *http.Request, s *general.Session) {
	DeleteCookie(w)
	DB.QueryRow("SELECT id FROM session WHERE uuid = ?", s.UUID).Scan(&s.ID)
	_, err = DB.Exec("DELETE FROM session WHERE id = ?", s.ID)
	*s = general.Session{UserID: 0}
	fmt.Println(s, "session after logout")
	http.Redirect(w, r, "/signin", 302)
}

//FileByte func for convert receive file - to fileByte
func FileByte(r *http.Request, typePhoto string) []byte {
	//check user photo || post photo
	r.ParseMultipartForm(10 << 20)
	file, _, err := r.FormFile("uploadfile")

	var defImg *os.File
	if err != nil {
		log.Println(err)
		//set default photo user
		if typePhoto == "user" {
			defImg, _ = os.Open("./utils/default-user.jpg")
		}
		file = defImg
	}
	defer file.Close()

	imgBytes, err := ioutil.ReadAll(file)

	if err != nil {
		log.Println(err)
	}
	return imgBytes
}

//AuthError show auth error, use js, fetch() query - use js func, showNotify() (signin handler	)
func AuthError(w http.ResponseWriter, r *http.Request, err error, text string, authType string) {

	fmt.Println(text, "notify msg")
	if authType == "default" {
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			m, _ := json.Marshal(text)
			w.Write(m)
			return
		}
		//send client - js receive value "success"
		w.WriteHeader(http.StatusOK)
		m, _ := json.Marshal(text)
		w.Write(m)
	}
}

//URLChecker function
func URLChecker(w http.ResponseWriter, r *http.Request, url string) bool {

	if r.URL.Path != url {
		RenderTemplate(w, "404page", http.StatusNotFound)
		return false
	}
	return true
}

//IsEmailValid function
func IsEmailValid(email string) bool {
	e := strings.Split(email, "@")
	Re := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
	if e[1] == "mail.kz" && Re.MatchString(email) {
		return true
	}
	return false
}

//IsPasswordValid function
func IsPasswordValid(s string) bool {
	var (
		hasMinLen  = false
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)
	if len(s) >= 7 {
		hasMinLen = true
	}
	for _, char := range s {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	return hasMinLen && hasUpper && hasLower && hasNumber && hasSpecial
}

//DeleteCookie func
func DeleteCookie(w http.ResponseWriter) {

	cookieDelete := http.Cookie{
		Name:     "_cookie",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: false,
	}
	http.SetCookie(w, &cookieDelete)

}
func GetUserID(cookie string) (uid int) {

	err = DB.QueryRow("SELECT id FROM session WHERE uuid = ?", cookie).Scan(&uid)

	if err != nil {
		log.Println(err)
	}
	return uid
}

//IsImage func
func IsImage(r *http.Request) []byte {

	f, _, _ := r.FormFile("uploadfile")
	photoFlag := false

	if f != nil {
		photoFlag = true
	}
	var imgBytes []byte

	if !photoFlag {
		imgBytes = []byte{0, 0}
	} else {
		imgBytes = FileByte(r, "post")
	}
	return imgBytes
}

//IsRegistered func
func IsRegistered(w http.ResponseWriter, r *http.Request, data string) bool {

	s := strings.Split(data, "@")
	field := "username"

	if len(s) == 2 {
		field = "email"
	}
	//check email by unique, if have same email
	count := 0
	var users []string
	var emailDB string

	err = DB.QueryRow("SELECT count(*) FROM users").Scan(&count)
	if err != nil {
		log.Println(err)
	}
	if count > 0 {

		checkUser, err := DB.Query("SELECT " + field + " FROM users")
		if err != nil {
			log.Println(err)
		}
		for checkUser.Next() {
			err = checkUser.Scan(&emailDB)
			if err != nil {
				log.Println(err.Error())
			}
			users = append(users, emailDB)
		}

		for _, v := range users {
			if v == data {
				log.Println(err)
				return true
			}
		}
	} else {
		return false
	}
	return false
}

//UpdateVoteNotify func
func UpdateVoteNotify(table string, toWhom, fromWhom, objID, voteType int) {

	fmt.Println(voteType, "vote TYPE", table)

	if table == "post" && toWhom != 0 {
		_, err = DB.Exec("UPDATE notify SET voteState=? WHERE comment_id=? AND post_id =? AND current_user_id=?  AND to_whom=?", voteType, 0, objID, fromWhom, toWhom)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(objID, fromWhom, toWhom, "update  Like/Dislike Post")

	} else if table == "comment" && toWhom != 0 {

		fmt.Println(objID, fromWhom, toWhom, "notify Update Vote Comment")
		_, err = DB.Exec("UPDATE notify SET voteState=? WHERE post_id=? AND  comment_id=? AND current_user_id=?  AND to_whom=?", voteType, 0, objID, fromWhom, toWhom)
		if err != nil {
			fmt.Println(err)
		}
	}
}

//SetVoteNotify func
func SetVoteNotify(table string, toWhom, fromWhom, objID int, voteLD bool) {

	voteState := 2
	if voteLD {
		voteState = 1
	}
	//putch(some fields), put(all fields),
	if table == "post" && toWhom != 0 {

		voteNotifyPreparePost, err := DB.Prepare(`INSERT INTO notify(post_id, current_user_id, voteState, created_time, to_whom, comment_id ) VALUES(?, ?, ?, ?, ?, ?)`)
		if err != nil {
			log.Println(err)
		}
		_, err = voteNotifyPreparePost.Exec(objID, fromWhom, voteState, time.Now(), toWhom, 0)
		if err != nil {
			log.Println(err, "Exec notify err")
		}
		fmt.Println(table, objID, fromWhom, toWhom, "notify Set Like/Dislike POST")
		defer voteNotifyPreparePost.Close()

	} else if table == "comment" && toWhom != 0 {

		fmt.Println(objID, fromWhom, toWhom, "notify Set Vote comment")

		voteNotifyPrepare, err := DB.Prepare(`INSERT INTO notify( post_id, current_user_id, voteState, created_time, to_whom, comment_id ) VALUES(?, ?, ?, ?, ?, ?)`)
		if err != nil {
			log.Println(err)
		}
		_, err = voteNotifyPrepare.Exec(0, fromWhom, voteState, time.Now(), toWhom, objID)
		if err != nil {
			log.Println(err)
		}
		defer voteNotifyPrepare.Close()
	}
}

//SetCommentNotify func by PostID
func SetCommentNotify(pid string, fromWhom, toWhom int, lid int64) {

	voteNotifyPrepare, err := DB.Prepare(`INSERT INTO notify(post_id, current_user_id, voteState, created_time, to_whom, comment_id ) VALUES(?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Println(err)
	}
	_, err = voteNotifyPrepare.Exec(pid, fromWhom, 0, time.Now(), toWhom, lid)
	if err != nil {
		log.Println(err)
	}
	defer voteNotifyPrepare.Close()
}

func ReSession(uid int, s *general.Session) {

	var sid int
	//first time enter signin system
	err = DB.QueryRow("SELECT id FROM session WHERE user_id=?", uid).Scan(&sid)
	if err != nil {
		log.Println(err, "no have session by uid")
		return
	}
	//same  email signin -> session, if have session -> drop session -> ReLogin
	_, err := DB.Exec("DELETE FROM session WHERE id = ?", sid)
	if err != nil {
		log.Println(err)
	}
	//set nil local session
	*s = general.Session{}
}
