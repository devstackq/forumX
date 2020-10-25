package routing

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/devstackq/ForumX/models"
	util "github.com/devstackq/ForumX/utils"
	"golang.org/x/crypto/bcrypt"
)

var (
	err error
	DB  *sql.DB
	API struct{ Message string }
)

//receive request, from client, query params, category ID, then query DB, depends catID, get Post this catID
func GetAllPosts(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/" && r.URL.Path != "/science" && r.URL.Path != "/love" && r.URL.Path != "/sapid" {
		util.DisplayTemplate(w, "404page", http.StatusNotFound)
		return
	}

	fv := models.Filter{
		Like:     r.FormValue("likes"),
		Date:     r.FormValue("date"),
		Category: r.FormValue("cats"),
	}

	posts, endpoint, category, err := fv.GetAllPost(r)

	if err != nil {
		log.Fatal(err)
	}

	util.DisplayTemplate(w, "header", util.IsAuth(r))

	if endpoint == "/" {
		util.DisplayTemplate(w, "index", posts)
	} else {
		//send category
		msg := []byte(fmt.Sprintf("<h2 id='category'> `Category: %s` </h2>", category))
		w.Header().Set("Content-Type", "application/json")
		w.Write(msg)
		util.DisplayTemplate(w, "catTemp", posts)
	}
}

//get 1 post by id
func GetPostById(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/post" {
		util.DisplayTemplate(w, "404page", http.StatusNotFound)
		return
	}
	id, _ := strconv.Atoi(r.FormValue("id"))
	p := models.Posts{ID: id}
	comments, post, err := p.GetPostById(r)

	if err != nil {
		log.Println(err)
	}
	util.DisplayTemplate(w, "header", util.IsAuth(r))
	util.DisplayTemplate(w, "posts", post)
	util.DisplayTemplate(w, "comment", comments)
}

//create post
func CreatePost(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/create/post" {
		util.DisplayTemplate(w, "404page", http.StatusNotFound)
		return
	}

	API.Message = ""

	switch r.Method {
	case "GET":
		util.DisplayTemplate(w, "header", util.IsAuth(r))
		util.DisplayTemplate(w, "create", &API.Message)
	case "POST":
		access, session := util.CheckForCookies(w, r)
		log.Println(access, "access status")
		if !access {
			http.Redirect(w, r, "/signin", 302)
			return
		}
		r.ParseMultipartForm(10 << 20)
		f, _, _ := r.FormFile("uploadfile")
		f2, _, _ := r.FormFile("uploadfile")
		categories, _ := r.Form["input"]

		post := models.Posts{
			Title:      r.FormValue("title"),
			Content:    r.FormValue("content"),
			Categories: categories,
			FileS:      f,
			FileI:      f2,
			Session:    session,
		}
		post.CreatePost(w, r)
	}
	http.Redirect(w, r, "/", http.StatusOK)
}

//update post
func UpdatePost(w http.ResponseWriter, r *http.Request) {

	if r.Method == "GET" {
		pid, _ := strconv.Atoi(r.URL.Query().Get("id"))
		p := models.Posts{}
		p.PostIDEdit = pid
		util.DisplayTemplate(w, "updatepost", p)
	}

	if r.Method == "POST" {

		access, _ := util.CheckForCookies(w, r)
		if !access {
			http.Redirect(w, r, "/signin", 302)
			return
		}
		imgBytes := util.FileByte(r)
		pid, _ := strconv.Atoi(r.FormValue("pid"))

		p := models.Posts{
			Title:   r.FormValue("title"),
			Content: r.FormValue("content"),
			Image:   imgBytes,
			ID:      pid,
		}

		err = p.UpdatePost()

		if err != nil {
			defer log.Println(err, "upd post err")
		}
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

//delete post
func DeletePost(w http.ResponseWriter, r *http.Request) {

	pid, _ := strconv.Atoi(r.URL.Query().Get("id"))
	p := models.Posts{ID: pid}

	access, _ := util.CheckForCookies(w, r)
	if !access {
		http.Redirect(w, r, "/signin", 302)
		return
	}

	err = p.DeletePost()

	if err != nil {
		panic(err.Error())
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

//create comment
func CreateComment(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/comment" {
		util.DisplayTemplate(w, "404page", http.StatusNotFound)
		return
	}

	if r.Method == "POST" {

		access, s := util.CheckForCookies(w, r)
		if !access {
			http.Redirect(w, r, "/signin", 302)
			return
		}

		DB.QueryRow("SELECT user_id FROM session WHERE uuid = ?", s.UUID).Scan(&s.UserID)

		pid, _ := strconv.Atoi(r.FormValue("curr"))
		comment := r.FormValue("comment-text")

		if util.CheckLetter(comment) {

			com := models.Comment{
				Content: comment,
				PostID:  pid,
				UserID:  s.UserID,
			}

			err = com.LeaveComment()

			if err != nil {
				log.Println(err.Error())
			}
		}
	}
	http.Redirect(w, r, "post?id="+r.FormValue("curr"), 301)
}

//profile current -> user page
func GetUserProfile(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/profile" {
		util.DisplayTemplate(w, "404page", http.StatusNotFound)
		return
	}

	if r.Method == "GET" {
		cookie, _ := r.Cookie("_cookie")
		//if userId now, createdPost uid equal -> show
		likedpost, posts, comments, user, err := models.GetUserProfile(r, w, cookie)
		if err != nil {
			panic(err)
		}

		util.DisplayTemplate(w, "header", util.IsAuth(r))
		util.DisplayTemplate(w, "profile", user)
		util.DisplayTemplate(w, "likedpost", likedpost)
		util.DisplayTemplate(w, "postuser", posts)
		util.DisplayTemplate(w, "commentuser", comments)

		//delete coookie db
		go func() {
			for now := range time.Tick(299 * time.Minute) {
				util.СheckCookieLife(now, cookie, w, r)
				//next logout each 300 min
				time.Sleep(299 * time.Minute)
			}
		}()
	}
}

//user page, other anyone
func GetAnotherProfile(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {

		uid := models.Users{Temp: r.FormValue("uid")}
		posts, user, err := uid.GetAnotherProfile(r)
		if err != nil {
			panic(err)
		}
		util.DisplayTemplate(w, "header", util.IsAuth(r))
		util.DisplayTemplate(w, "user", user)
		util.DisplayTemplate(w, "postuser", posts)
	}
}

//update profile
func UpdateProfile(w http.ResponseWriter, r *http.Request) {

	if r.Method == "GET" {
		util.DisplayTemplate(w, "header", util.IsAuth(r))
		util.DisplayTemplate(w, "updateuser", "")
	}

	if r.Method == "POST" {

		access, s := util.CheckForCookies(w, r)
		if !access {
			http.Redirect(w, r, "/signin", 302)
			return
		}

		imgBytes := util.FileByte(r)

		DB.QueryRow("SELECT user_id FROM session WHERE uuid = ?", s.UUID).
			Scan(&s.UserID)

		is, _ := strconv.Atoi(r.FormValue("age"))

		p := models.Users{
			FullName: r.FormValue("fullname"),
			Age:      is,
			Sex:      r.FormValue("sex"),
			City:     r.FormValue("city"),
			Image:    imgBytes,
			ID:       s.UserID,
		}

		err = p.UpdateProfile()

		if err != nil {
			panic(err.Error())
		}
	}
	http.Redirect(w, r, "/profile", http.StatusFound)
}

//search
func Search(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/search" {
		util.DisplayTemplate(w, "404page", http.StatusNotFound)
		return
	}

	if r.Method == "GET" {
		util.DisplayTemplate(w, "search", http.StatusFound)
	}

	if r.Method == "POST" {

		foundPosts, err := models.Search(w, r)

		if err != nil {
			panic(err)
		}
		util.DisplayTemplate(w, "header", util.IsAuth(r))
		util.DisplayTemplate(w, "index", foundPosts)
	}
}
here pause
//signup system
func Signup(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/signup" {
		util.DisplayTemplate(w, "404page", http.StatusNotFound)
		return
	}

	msg := models.API

	if r.Method == "GET" {
		util.DisplayTemplate(w, "signup", &msg)
	}

	if r.Method == "POST" {

		fn := r.FormValue("fullname")
		e := r.FormValue("email")
		p := r.FormValue("password")
		a := r.FormValue("age")
		s := r.FormValue("sex")
		c := r.FormValue("city")

		imgBytes := util.FileByte(r)
		hash, err := bcrypt.GenerateFromPassword([]byte(p), 8)
		if err != nil {
			panic(err)
		}

		//check email by unique, if have same email
		checkEmail, err := DB.Query("SELECT email FROM users")
		if err != nil {
			panic(err)
		}

		all := []models.Users{}

		for checkEmail.Next() {
			user := models.Users{}
			var email string
			err = checkEmail.Scan(&email)
			if err != nil {
				panic(err.Error)
			}

			user.Email = email
			all = append(all, user)
		}

		for _, v := range all {
			if v.Email == e {
				API.Message = "Not unique email lel"
				util.DisplayTemplate(w, "signup", &API.Message)
				return
			}
		}

		_, err = DB.Exec("INSERT INTO users( full_name, email, password, age, sex, city, image) VALUES (?, ?, ?, ?, ?, ?, ?)",
			fn, e, hash, a, s, c, imgBytes)

		if err != nil {
			panic(err.Error())
		}

		http.Redirect(w, r, "/signin", 301)
	}
}

//signin system
func Signin(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/signin" {
		util.DisplayTemplate(w, "404page", http.StatusNotFound)
		return
	}
	r.Header.Add("Accept", "text/html")
	r.Header.Add("User-Agent", "MSIE/15.0")

	API.Message = ""

	if r.Method == "GET" {
		util.DisplayTemplate(w, "signin", &API.Message)
	}

	if r.Method == "POST" {
		var person models.Users
		//b, _ := ioutil.ReadAll(r.Body)
		err := json.NewDecoder(r.Body).Decode(&person)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		fmt.Println(person, "person value")

		if person.Type == "default" {

			fmt.Println(" default auth")
			models.Signin(w, r, person.Email, person.Password)
			http.Redirect(w, r, "/profile", 200)

		} else if person.Type == "google" {
			fmt.Println("todo google auth")
			http.Redirect(w, r, "/profile", http.StatusFound)
		} else if person.Type == "github" {
			fmt.Println("todo github")
			http.Redirect(w, r, "/profile", http.StatusFound)
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}
}

// Logout
func Logout(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/logout" {
		util.DisplayTemplate(w, "404page", http.StatusNotFound)
		return
	}
	if r.Method == "GET" {
		models.Logout(w, r)

		http.Redirect(w, r, "/signin", 302)
	}
}

//like dislike post
func LostVotes(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/votes" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	access, s := util.CheckForCookies(w, r)
	if !access {
		http.Redirect(w, r, "/signin", 302)
		return
	}
	//c, _ := r.Cookie("_cookie")
	//s := models.Session{UUID: c.Value}

	DB.QueryRow("SELECT user_id FROM session WHERE uuid = ?", s.UUID).
		Scan(&s.UserID)

	pid := r.URL.Query().Get("id")
	lukas := r.FormValue("lukas")
	diskus := r.FormValue("diskus")

	if r.Method == "POST" {

		if lukas == "1" {
			//check if not have post and user lost vote this post
			//1 like or 1 dislike 1 user lost 1 post, get previus value and +1
			var p, u int
			err = DB.QueryRow("SELECT post_id, user_id FROM likes WHERE post_id=? AND user_id=?", pid, s.UserID).Scan(&p, &u)

			if p == 0 && u == 0 {

				oldlike := 0
				err = DB.QueryRow("SELECT count_like FROM posts WHERE id=?", pid).Scan(&oldlike)
				nv := oldlike + 1
				_, err = DB.Exec("UPDATE  posts SET count_like = ? WHERE id= ?", nv, pid)
				if err != nil {
					panic(err)
				}

				_, err = DB.Exec("INSERT INTO likes(post_id, user_id, state_id) VALUES( ?, ?, ?)", pid, s.UserID, 1)
				if err != nil {
					panic(err)
				}
			}
		}

		if diskus == "1" {

			var p, u int
			err = DB.QueryRow("SELECT post_id, user_id FROM likes WHERE post_id=? AND user_id=?", pid, s.UserID).Scan(&p, &u)

			if p == 0 && u == 0 {

				oldlike := 0
				err = DB.QueryRow("select count_dislike from posts where id=?", pid).Scan(&oldlike)
				nv := oldlike + 1
				_, err = DB.Exec("UPDATE  posts SET count_dislike = ? WHERE id= ?", nv, pid)
				if err != nil {
					panic(err)
				}
				_, err = DB.Exec("INSERT INTO likes(post_id, user_id, state_id) VALUES( ?, ?, ?)", pid, s.UserID, 0)

				if err != nil {
					panic(err)
				}
			}
		}
	}
	http.Redirect(w, r, "post?id="+pid, 301)
}

func LostVotesComment(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/votes/comment" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	access, s := util.CheckForCookies(w, r)
	if !access {
		http.Redirect(w, r, "/signin", 302)
		return
	}
	//c, _ := r.Cookie("_cookie")
	//s := models.Session{UUID: c.Value}
	DB.QueryRow("SELECT user_id FROM session WHERE uuid = ?", s.UUID).
		Scan(&s.UserID)

	cid := r.URL.Query().Get("cid")
	comdis := r.FormValue("comdis")
	comlike := r.FormValue("comlike")

	pidc := r.FormValue("pidc")

	if r.Method == "POST" {

		if comlike == "1" {

			var c, u int
			err = DB.QueryRow("SELECT comment_id, user_id FROM likes WHERE comment_id=? AND user_id=?", cid, s.UserID).Scan(&c, &u)

			if c == 0 && u == 0 {

				oldlike := 0
				err = DB.QueryRow("SELECT com_like FROM comments WHERE id=?", cid).Scan(&oldlike)
				nv := oldlike + 1

				_, err = DB.Exec("UPDATE  comments SET com_like = ? WHERE id= ?", nv, cid)

				if err != nil {
					panic(err)
				}

				_, err = DB.Exec("INSERT INTO likes(comment_id, user_id) VALUES( ?, ?)", cid, s.UserID)
				if err != nil {
					panic(err)
				}
			}
		}

		if comdis == "1" {

			var c, u int
			err = DB.QueryRow("SELECT comment_id, user_id FROM likes WHERE comment_id=? AND user_id=?", cid, s.UserID).Scan(&c, &u)

			if c == 0 && u == 0 {

				oldlike := 0
				err = DB.QueryRow("SELECT com_dislike FROM comments WHERE id=?", cid).Scan(&oldlike)
				nv := oldlike + 1

				_, err = DB.Exec("UPDATE  comments SET com_dislike = ? WHERE id= ?", nv, cid)

				if err != nil {
					panic(err)
				}

				_, err = DB.Exec("INSERT INTO likes(comment_id, user_id) VALUES( ?, ?)", cid, s.UserID)
				if err != nil {
					panic(err)
				}
			}
		}
		http.Redirect(w, r, "/post?id="+pidc, 301)
	}
}

//Likes table, filed posrid, userid, state_id
// 0,1,2 if state ==0, 1 || 2,
// next btn, if 1 == 1, state =0
