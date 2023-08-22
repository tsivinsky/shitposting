package db

import (
	"database/sql"
	"time"
)

type Post struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Slug    string `json:"slug"`
	Body    string `json:"body"`
	Created string `json:"created"`
}

func FindPosts(pool *sql.DB) ([]Post, error) {
	rows, err := pool.Query("select * from posts order by created desc;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post

	for rows.Next() {
		post := new(Post)
		err = rows.Scan(&post.ID, &post.Title, &post.Slug, &post.Body, &post.Created)
		if err != nil {
			return nil, err
		}

		posts = append(posts, *post)
	}

	return posts, nil
}

func FindPostBySlug(pool *sql.DB, slug string) (*Post, error) {
	rows, err := pool.Query("select * from posts where slug = $1;", slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rows.Next()

	var post Post
	err = rows.Scan(&post.ID, &post.Title, &post.Slug, &post.Body, &post.Created)
	if err != nil {
		return nil, err
	}

	return &post, nil
}

func CreatePost(pool *sql.DB, title, slug, body string) error {
	created := time.Now().Format(time.RFC3339)
	_, err := pool.Exec("insert into posts (title, slug, body, created) values ($1, $2, $3, $4);", title, slug, body, created)
	if err != nil {
		return err
	}

	return nil
}
