package models

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/devstackq/ForumX/model"
	util "github.com/devstackq/ForumX/utils"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	err                          error
	DB                           *sql.DB
	rows                         *sql.Rows
	id, creatorID, like, dislike int
	content, title               string
	createdTime                  time.Time
	image                        []byte
	postID                       int
	userID                       int
	post                         Posts
	comment                      Comment
)

type Users struct {
	ID          int
	FullName    string
	Email       string
	Password    string
	IsAdmin     bool
	Age         int
	Sex         string
	CreatedTime time.Time
	City        string
	Image       []byte
	ImageHtml   string
	Role        string
	SVG         bool
	Type        string
	Temp        string
}

type Category struct {
	ID     int
	Name   string
	UserID int
}

type Filter struct {
	Category string
	Like     string
	Date     string
}

type PostCategory struct {
	PostID   int64
	Category string
}

//comment ID -> foreign key -> postID
type Comments struct {
	ID             int
	Commentik      string
	PostID         int
	UserID         int
	CreatedTime    time.Time
	AuthorComment  string
	CommentLike    int
	CommentDislike int
}

var API struct {
	Authenticated bool
}

type Likes struct {
	ID      int
	Like    int
	Dislike int
	PostID  int
	UserID  int
	Voted   bool
}
type Notify struct {
	Message string
}

func (pcb *PostCategory) CreateBridge() error {
	_, err := DB.Exec("INSERT INTO post_cat_bridge (post_id, category) VALUES (?, ?)",
		pcb.PostID, pcb.Category)
	if err != nil {
		return err
	}
	return nil
}

//update profile
func (u *Users) UpdateProfile() error {

	_, err := DB.Exec("UPDATE  users SET full_name=?, age=?, sex=?, city=?, image=? WHERE id =?",
		u.FullName, u.Age, u.Sex, u.City, u.Image, u.ID)
	if err != nil {
		return err
	}
	return nil
}

//siginin
func Signin(w http.ResponseWriter, r *http.Request, email, password string) {

	u := DB.QueryRow("SELECT id, password FROM users WHERE email=?", email)

	var user Users
	var err error
	//check pwd, if not correct, error
	err = u.Scan(&user.ID, &user.Password)
	if err != nil {
		authError(w, err, "user not found")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		authError(w, err, "password incorrect")
		return
	}
	//get user by Id, and write session struct
	s := model.Session{
		UserID: user.ID,
	}
	uuid := uuid.Must(uuid.NewV4(), err).String()
	if err != nil {
		authError(w, err, "uuid trouble")
		return
	}

	//create uuid and set uid DB table session by userid,
	_, err = DB.Exec("INSERT INTO session(uuid, user_id) VALUES (?, ?)", uuid, s.UserID)
	if err != nil {
		authError(w, err, "the user is already in the system")
		return
	}

	// get user in info by session Id
	err = DB.QueryRow("SELECT id, uuid FROM session WHERE user_id = ?", s.UserID).Scan(&s.ID, &s.UUID)
	if err != nil {
		authError(w, err, "not find user from session")
		return
	}
	//set cookie 9128ueq9widjaisdh238yrhdeiuwandijsan
	// Crete post -> Cleint cookie == session, Userd
	cookie := http.Cookie{
		Name:     "_cookie",
		Value:    s.UUID,
		Path:     "/",
		Expires:  time.Now().Add(300 * time.Minute),
		HttpOnly: false,
	}
	http.SetCookie(w, &cookie)
	authError(w, nil, "success")
}

func authError(w http.ResponseWriter, err error, text string) {
	fmt.Println(text, "errka")
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		m, _ := json.Marshal(text)
		w.Write(m)
		return
	} else {
		w.WriteHeader(200)
		m, _ := json.Marshal(text)
		w.Write(m)
	}
}

func Logout(w http.ResponseWriter, r *http.Request) {

	cookie, err := r.Cookie("_cookie")
	if err != nil {
		fmt.Println(err, "cookie err")
	}
	//add cookie -> fields uuid
	s := model.Session{UUID: cookie.Value}
	//get ssesion id, by local struct uuid
	DB.QueryRow("SELECT id FROM session WHERE uuid = ?", s.UUID).
		Scan(&s.ID)
	fmt.Println(s.ID, "id del session")
	//delete session by id session
	_, err = DB.Exec("DELETE FROM session WHERE id = ?", s.ID)

	if err != nil {
		panic(err)
	}

	// then delete cookie from client
	cDel := http.Cookie{
		Name:     "_cookie",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: false,
	}
	http.SetCookie(w, &cDel)
	http.Redirect(w, r, "/", http.StatusOK)

}

//get profile by id
func GetUserProfile(r *http.Request, w http.ResponseWriter, cookie *http.Cookie) ([]Posts, []Posts, []Comment, Users, error) {

	//time.AfterFunc(10, checkCookieLife(cookie, w, r))
	s := model.Session{UUID: cookie.Value}
	u := Users{}
	DB.QueryRow("SELECT user_id FROM session WHERE uuid = ?", s.UUID).Scan(&s.UserID)
	lps := []Likes{}

	//count dislike equal 0 - add query
	lp, err := DB.Query("select post_id from likes where user_id =? and state_id =?", s.UserID, 1)
	defer lp.Close()
	for lp.Next() {
		l := Likes{}
		var lpid int
		err = lp.Scan(&lpid)
		l.PostID = lpid
		lps = append(lps, l)
	}
	fmt.Println(lps, "Like post id")

	err = DB.QueryRow("SELECT * FROM users WHERE id = ?", s.UserID).Scan(&u.ID, &u.FullName, &u.Email, &u.Password, &u.IsAdmin, &u.Age, &u.Sex, &u.CreatedTime, &u.City, &u.Image)
	if u.Image[0] == 60 {
		u.SVG = true
	}

	encStr := base64.StdEncoding.EncodeToString(u.Image)
	u.ImageHtml = encStr

	var likedpost *sql.Rows
	postsL := []Posts{}

	var can []int

	for _, v := range lps {
		can = append(can, v.PostID)
	}

	//unique liked post by user
	fin := util.IsUnique(can)
	//accum liked post

	for _, v := range fin {
		//get each liked post by ID, then likedpost, put array post
		//		likedpost, err = DB.Query("SELECT * FROM posts WHERE id=? ", v)
		fmt.Println(v, "pid")
		//count_dislike не
		likedpost, err = DB.Query("SELECT * FROM posts WHERE id=? and count_like > 0", v)
		if err != nil {
			log.Println(err)
		}
		for likedpost.Next() {
			err = likedpost.Scan(&id, &title, &content, &creatorID, &createdTime, &image, &like, &dislike)
			if err != nil {
				panic(err.Error)
			}
			post = appendPost(id, title, content, creatorID, image, like, dislike, s.UserID, createdTime)
			postsL = append(postsL, post)
		}
	}
	//create post current user
	psu, err := DB.Query("SELECT * FROM posts WHERE creator_id=?", s.UserID)
	//defer psu.Close()
	var postCr Posts
	postsX := []Posts{}

	//todo get uniq post - created post
	for psu.Next() {
		//here
		err = psu.Scan(&id, &title, &content, &creatorID, &createdTime, &image, &like, &dislike)
		if err != nil {
			log.Println(err.Error())
		}
		//uniq create post
		//post.AuthorForPost = s.UserID
		fmt.Println(s.UserID, "uid count created post", id, "pid")

		postCr = appendPost(id, title, content, creatorID, image, like, dislike, s.UserID, createdTime)
		postsX = append(postsX, postCr)
	}

	csu, err := DB.Query("SELECT * FROM comments WHERE user_idx=?", s.UserID)
	var comments []Comment
	defer csu.Close()

	for csu.Next() {

		err = csu.Scan(&id, &content, &postID, &userID, &createdTime, &like, &dislike)
		err = DB.QueryRow("SELECT title FROM posts WHERE id = ?", postID).Scan(&title)
		if err != nil {
			panic(err.Error)
		}

		comment = appendComment(id, content, postID, userID, createdTime, like, dislike, title)
		comments = append(comments, comment)
	}

	if err != nil {
		return nil, nil, nil, u, err
	}

	return postsL, postsX, comments, u, nil
}

//get other user, posts
func (user *Users) GetAnotherProfile(r *http.Request) ([]Posts, Users, error) {

	userQR := DB.QueryRow("SELECT * FROM users WHERE id = ?", user.Temp)

	u := Users{}
	postsU := []Posts{}

	err = userQR.Scan(&u.ID, &u.FullName, &u.Email, &u.Password, &u.IsAdmin, &u.Age, &u.Sex, &u.CreatedTime, &u.City, &u.Image)
	if u.Image[0] == 60 {
		u.SVG = true
	}
	encStr := base64.StdEncoding.EncodeToString(u.Image)
	u.ImageHtml = encStr
	psu, err := DB.Query("SELECT * FROM posts WHERE creator_id=?", u.ID)

	defer psu.Close()

	var image []byte

	for psu.Next() {
		err = psu.Scan(&id, &title, &content, &creatorID, &createdTime, &image, &like, &dislike)

		if err != nil {
			panic(err.Error)
		}
		post = appendPost(id, title, content, creatorID, image, like, dislike, 0, createdTime)
		postsU = append(postsU, post)
	}
	if err != nil {
		return nil, u, err
	}
	return postsU, u, nil
}

//search
func Search(w http.ResponseWriter, r *http.Request) ([]Posts, error) {

	psu, err := DB.Query("SELECT * FROM posts WHERE title LIKE ?", "%"+r.FormValue("search")+"%")
	defer psu.Close()
	var posts []Posts

	for psu.Next() {

		err = psu.Scan(&id, &title, &content, &creatorID, &createdTime, &image, &like, &dislike)
		if err != nil {
			log.Println(err.Error())
		}
		post = appendPost(id, title, content, creatorID, image, like, dislike, 0, createdTime)
		posts = append(posts, post)
	}

	if err != nil {
		return nil, err
	}
	return posts, nil
}

//appendPost each post put value from Db
func appendPost(id int, title, content string, creatorID int, image []byte, like, dislike, authorID int, createdTime time.Time) Posts {

	post = Posts{
		ID:            id,
		Title:         title,
		Content:       content,
		CreatorID:     creatorID,
		Image:         image,
		CountLike:     like,
		CountDislike:  dislike,
		AuthorForPost: authorID,
		CreatedTime:   createdTime,
	}
	return post
}
func appendComment(id int, content string, postID, userID int, createdTime time.Time, like, dislike int, titlePost string) Comment {

	comment = Comment{
		ID:          id,
		Content:     content,
		PostID:      postID,
		UserID:      userID,
		CreatedTime: createdTime,
		Like:        like,
		Dislike:     dislike,
		TitlePost:   titlePost,
	}
	return comment
}
