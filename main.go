package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/tsivinsky/goenv"

	_ "github.com/lib/pq"
)

type Post struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Slug  string `json:"slug"`
	Body  string `json:"body"`
}

type Env struct {
	DBUser     string `env:"POSTGRES_USER,required"`
	DBPassword string `env:"POSTGRES_PASSWORD,required"`
	DBName     string `env:"POSTGRES_DB,required"`
	DBHost     string `env:"DB_HOST,required"`
}

var pool *sql.DB

func main() {
	env := new(Env)
	goenv.MustLoad(env)

	dsn := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable", env.DBUser, env.DBPassword, env.DBHost, env.DBName)

	pool, _ = sql.Open("postgres", dsn)
	defer pool.Close()

	pool.SetConnMaxLifetime(0)
	pool.SetMaxIdleConns(3)
	pool.SetMaxOpenConns(3)

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("./views/index.html")
		if err != nil {
			renderErrorPage(w, "error happened, sorry")
			return
		}

		posts, err := findPosts()
		if err != nil {
			fmt.Printf("err: %v\n", err)
			renderErrorPage(w, "couldn't find posts")
			return
		}

		tmpl.Execute(w, struct {
			Posts []Post
		}{
			Posts: posts,
		})
	})

	mux.HandleFunc("/p/", func(w http.ResponseWriter, r *http.Request) {
		slug := strings.Split(r.URL.Path, "/")[1:][1]

		post, err := findPostBySlug(slug)
		if err != nil {
			renderErrorPage(w, "couldn't find post")
			return
		}

		if post == nil {
			renderErrorPage(w, "post not found")
			return
		}

		tmpl, err := template.ParseFiles("./views/post.html")
		if err != nil {
			renderErrorPage(w, "error happened, sorry")
			return
		}

		tmpl.Execute(w, &post)
	})

	mux.HandleFunc("/post", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			tmpl, err := template.ParseFiles("./views/create-post.html")
			if err != nil {
				renderErrorPage(w, "error happened, sorry")
				return
			}

			tmpl.Execute(w, nil)
			return
		}

		if r.Method == "POST" {
			title := r.FormValue("title")
			slug := r.FormValue("slug")
			body := r.FormValue("body")

			err := createPost(title, slug, body)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				renderErrorPage(w, "error while creating post")
				return
			}

			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
	})

	log.Fatal(http.ListenAndServe(":5000", mux))
}

func findPosts() ([]Post, error) {
	rows, err := pool.Query("select * from posts;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post

	for rows.Next() {
		post := new(Post)
		err = rows.Scan(&post.ID, &post.Title, &post.Slug, &post.Body)
		if err != nil {
			return nil, err
		}

		posts = append(posts, *post)
	}

	return posts, nil
}

func findPostBySlug(slug string) (*Post, error) {
	rows, err := pool.Query("select * from posts where slug = $1;", slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rows.Next()

	var post Post
	err = rows.Scan(&post.ID, &post.Title, &post.Slug, &post.Body)
	if err != nil {
		return nil, err
	}

	return &post, nil
}

func createPost(title, slug, body string) error {
	_, err := pool.Exec("insert into posts (title, slug, body) values ($1, $2, $3);", title, slug, body)
	if err != nil {
		return err
	}

	return nil
}

func renderErrorPage(w http.ResponseWriter, message string) error {
	tmpl, err := template.ParseFiles("./views/error.html")
	if err != nil {
		return err
	}

	return tmpl.Execute(w, map[string]string{
		"Error": message,
	})
}