package models

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/devstackq/ForumX/model"
	util "github.com/devstackq/ForumX/utils"
)

type Posts struct {
	ID            int
	Title         string
	Content       string
	CreatorID     int
	CreatedTime   time.Time
	Endpoint      string
	FullName      string
	CategoryName  string
	Image         []byte
	ImageHtml     string
	PostIDEdit    int
	AuthorForPost int
	CountLike     int
	CountDislike  int
	SVG           bool
	PBGID         int
	PBGPostID     int
	PBGCategory   string
	LastPostId    int
	FileS         multipart.File
	FileI         multipart.File
	Session       model.Session
	Categories    []string
	Temp          string
}

//func GetAllPost(r *http.Request, like, date, category string) ([]Posts, string, string, error) {
func (f *Filter) GetAllPost(r *http.Request) ([]Posts, string, string, error) {

	var post Posts
	//send from controlle, then check-> then send model

	var leftJoin bool
	var arrPosts []Posts

	switch r.URL.Path {
	//check what come client, cats, and filter by like, date and cats
	case "/":
		leftJoin = false
		post.Endpoint = "/"
		if f.Date == "asc" {
			rows, err = DB.Query("SELECT * FROM posts  ORDER BY created_time ASC LIMIT 6")
		} else if f.Date == "desc" {
			rows, err = DB.Query("SELECT * FROM posts  ORDER BY created_time DESC LIMIT 6")
		} else if f.Like == "like" {
			rows, err = DB.Query("SELECT * FROM posts  ORDER BY count_like DESC LIMIT 6")
		} else if f.Like == "dislike" {
			rows, err = DB.Query("SELECT * FROM posts  ORDER BY count_dislike DESC LIMIT 6")
		} else if f.Category != "" {
			leftJoin = true
			rows, err = DB.Query("SELECT  * FROM posts  LEFT JOIN post_cat_bridge  ON post_cat_bridge.post_id = posts.id   WHERE category=? ORDER  BY created_time  DESC LIMIT 6", f.Category)
		} else {
			rows, err = DB.Query("SELECT * FROM posts  ORDER BY created_time DESC LIMIT 6")
		}

	case "/science":
		leftJoin = true
		post.Endpoint = "/science"
		post.Temp = "Science"
		rows, err = DB.Query("SELECT * FROM posts  LEFT JOIN post_cat_bridge  ON post_cat_bridge.post_id = posts.id   WHERE category=?  ORDER  BY created_time  DESC LIMIT 4", "science")
	case "/love":
		post.Temp = "Love"
		leftJoin = true
		post.Endpoint = "/love"
		rows, err = DB.Query("SELECT  * FROM posts  LEFT JOIN post_cat_bridge  ON post_cat_bridge.post_id = posts.id  WHERE category=?   ORDER  BY created_time  DESC LIMIT 4", "love")
	case "/sapid":
		post.Temp = "Sapid"
		leftJoin = true
		post.Endpoint = "/sapid"
		rows, err = DB.Query("SELECT  * FROM posts  LEFT JOIN post_cat_bridge  ON post_cat_bridge.post_id = posts.id  WHERE category=?  ORDER  BY created_time  DESC LIMIT 4", "sapid")
	}

	defer rows.Close()
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	for rows.Next() {
		post := Posts{}
		if leftJoin {
			if err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.CreatorID, &post.CreatedTime, &post.Image, &post.CountLike, &post.CountDislike, &post.PBGID, &post.PBGPostID, &post.PBGCategory); err != nil {
				fmt.Println(err)
			}
		} else {
			if err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.CreatorID, &post.CreatedTime, &post.Image, &post.CountLike, &post.CountDislike); err != nil {
				fmt.Println(err)
			}
		}

		arrPosts = append(arrPosts, post)
	}
	//	fmt.Println(arrayPosts, "osts all")
	return arrPosts, post.Endpoint, post.Temp, nil
}

//create post
// func (p *Posts) CreatePost() (int64, error) {
// 	db, err := DB.Exec("INSERT INTO posts (title, content, creator_id,  image) VALUES ( ?,?, ?, ?)",
// 		p.Title, p.Content, p.CreatorID, p.Image)
// 	if err != nil {
// 		return 0, err
// 	}
// 	//DB.QueryRow("SELECT id FROM posts").Scan(&p.La)
// 	last, err := db.LastInsertId()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	return last, nil
// }

//update post
func (p *Posts) UpdatePost() error {

	_, err := DB.Exec("UPDATE  posts SET title=?, content=?, image=? WHERE id =?",
		p.Title, p.Content, p.Image, p.PostIDEdit)

	if err != nil {
		return err
	}
	return nil
}

//delete post
func (p *Posts) DeletePost() error {
	_, err := DB.Exec("DELETE FROM  posts  WHERE id =?", p.PostIDEdit)
	if err != nil {
		return err
	}
	return nil
}

func (data *Posts) CreatePost(w http.ResponseWriter, r *http.Request) {

	//try default photo user or post
	// fImg, err := os.Open("./1553259670.jpg")

	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(1)
	// }
	// defer fImg.Close()

	// imgInfo, err := fImg.Stat()
	// if err != nil {
	// 	fmt.Println(err, "stats")
	// 	os.Exit(1)
	// }

	// var size int64 = imgInfo.Size()
	// fmt.Println(size, "size")
	// byteArr := make([]byte, size)

	// read file into bytes
	// buffer := bufio.NewReader(fImg)
	// _, err = buffer.Read(byteArr)
	//defer fImg.Close()

	var fileBytes []byte

	var buff bytes.Buffer
	fileSize, _ := buff.ReadFrom(data.FileS)
	defer data.FileS.Close()

	if fileSize < 20000000 {
		//file2, _, err := r.FormFile("uploadfile")
		if err != nil {
			log.Fatal(err)
		}
		fileBytes, err = ioutil.ReadAll(data.FileI)
	} else {
		fmt.Print("file more 20mb")
		//message  client send
		util.DisplayTemplate(w, "header", util.IsAuth(r))
		util.DisplayTemplate(w, "create", "Large file, more than 20mb")
	}

	DB.QueryRow("SELECT user_id FROM session WHERE uuid = ?", data.Session.UUID).Scan(&data.Session.UserID)
	//check empty values
	if util.CheckLetter(data.Title) && util.CheckLetter(data.Content) {

		db, err := DB.Exec("INSERT INTO posts (title, content, creator_id,  image) VALUES ( ?,?, ?, ?)",
			data.Title, data.Content, data.Session.UserID, fileBytes)
		if err != nil {
			log.Println(err)
		}
		//DB.QueryRow("SELECT id FROM posts").Scan(&p.La)
		last, err := db.LastInsertId()
		if err != nil {
			log.Fatal(err)
		}
		//return last, nil
		//insert cat_post_bridge value

		if len(data.Categories) == 1 {
			pcb := PostCategory{
				PostID:   last,
				Category: data.Categories[0],
			}
			err = pcb.CreateBridge()
			if err != nil {
				log.Println(err)
			}
		} else if len(data.Categories) > 1 {
			//loop
			for _, v := range data.Categories {
				pcb := PostCategory{
					PostID:   last,
					Category: v,
				}
				err = pcb.CreateBridge()
				if err != nil {
					log.Println(err)
				}
			}
		}
		w.WriteHeader(http.StatusCreated)
		http.Redirect(w, r, "/", http.StatusOK)
	} else {
		util.DisplayTemplate(w, "header", util.IsAuth(r))
		util.DisplayTemplate(w, "create", "Empty title or content")
	}
}

//link to COmments struct, then call func(r), return arr comments, post, err
func (post *Posts) GetPostById(r *http.Request) ([]Comment, Posts, error) {

	p := Posts{}
	//take from all post, only post by id, then write data struct Post
	DB.QueryRow("SELECT * FROM posts WHERE id = ?", post.ID).Scan(&p.ID, &p.Title, &p.Content, &p.CreatorID, &p.CreatedTime, &p.Image, &p.CountLike, &p.CountDislike)
	p.CreatedTime.Format(time.RFC1123)
	//write values from tables Likes, and write data table Post fileds like, dislikes
	//[]byte -> encode string, client render img base64
	if len(p.Image) > 0 {
		if p.Image[0] == 60 {
			p.SVG = true
		}
	}

	encodedString := base64.StdEncoding.EncodeToString(p.Image)
	p.ImageHtml = encodedString

	//creator post
	DB.QueryRow("SELECT full_name FROM users WHERE id = ?", p.CreatorID).Scan(&p.FullName)
	//get category post
	//DB.QueryRow("SELECT category FROM post_cat_bridge WHERE post_id=?", p.ID).Scan(&p.CategoryName)
	//get all comments from post
	stmp, err := DB.Query("SELECT * FROM comments WHERE  post_id =?", p.ID)
	if err != nil {
		log.Fatal(err)
	}
	defer stmp.Close()
	//write each fileds inside Comment struct -> then  append Array Comments
	var comments []Comment

	for stmp.Next() {
		comment := Comment{}
		var id, postID, userID, comLike, comDislike int
		var content string
		var myTime time.Time
		err = stmp.Scan(&id, &content, &postID, &userID, &myTime, &comLike, &comDislike)
		if err != nil {
			panic(err.Error)
		}

		comment = Comment{
			ID:          id,
			Content:     content,
			PostID:      postID,
			UserID:      userID,
			CreatedTime: createdTime,
			Like:        like,
			Dislike:     dislike,
		}
		//comment = util.AppendComment(id, content, postID, userID, createdTime, like, dislike)
		comments = append(comments, comment)

		DB.QueryRow("SELECT full_name FROM users WHERE id = ?", userID).Scan(&comment.Author)
	}

	if err != nil {
		return nil, p, err
	}
	return comments, p, nil
}
