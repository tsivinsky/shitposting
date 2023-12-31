package main

import (
	"blog/db"
	"database/sql"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/tsivinsky/goenv"

	_ "github.com/lib/pq"
)

type Env struct {
	DBUser     string `env:"POSTGRES_USER,required"`
	DBPassword string `env:"POSTGRES_PASSWORD,required"`
	DBName     string `env:"POSTGRES_DB,required"`
	DBHost     string `env:"DB_HOST,required"`
	User       string `env:"USER,required"`
	Password   string `env:"PASSWORD,required"`
}

var pool *sql.DB

var env = new(Env)

func main() {
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

		posts, err := db.FindPosts(pool)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			renderErrorPage(w, "couldn't find posts")
			return
		}

		tmpl.Execute(w, struct {
			Posts []db.Post
		}{
			Posts: posts,
		})
	})

	mux.HandleFunc("/p/", func(w http.ResponseWriter, r *http.Request) {
		slug := strings.Split(r.URL.Path, "/")[1:][1]

		post, err := db.FindPostBySlug(pool, slug)
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
			a := r.Header.Get("Authorization")
			if a == "" {
				w.Header().Set("WWW-Authenticate", "basic realm=Restricted")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			credsCorrect, err := validateAuth(a)
			if err != nil {
				renderErrorPage(w, err.Error())
				return
			}

			if !credsCorrect {
				http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
				return
			}

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

			err := db.CreatePost(pool, title, slug, body)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				renderErrorPage(w, "error while creating post")
				return
			}

			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
	})

	log.Fatal(http.ListenAndServe(":9090", mux))
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

func validateAuth(auth string) (bool, error) {
	s := strings.Split(auth, " ")
	data, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		return false, err
	}
	decoded := string(data)

	a := strings.Split(decoded, ":")
	user := a[0]
	password := a[1]

	return user == env.User && password == env.Password, nil
}
