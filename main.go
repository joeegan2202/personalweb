package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"strings"

	"crawshaw.io/sqlite/sqlitex"
)

//go:embed index.css index.html blog.css blog.html avatar.jpg
var f embed.FS

var dbpool *sqlitex.Pool

func main() {
	var err error
	dbpool, err = sqlitex.Open("file:posts.db", 0, 10)
	if err != nil {
		log.Fatalf("Fatal Error: %s\n", err.Error())
	}

	assertSchemas()

	http.Handle("/", http.FileServer(http.FS(f)))

	http.Handle("/blog/", http.StripPrefix("/blog/", http.HandlerFunc(blog)))

	log.Fatal(http.ListenAndServe(":80", nil))
}

func blog(w http.ResponseWriter, r *http.Request) {
	conn := dbpool.Get(r.Context())

	if conn == nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error requesting database connection!"))
		return
	}

	defer dbpool.Put(conn)

	if r.URL.EscapedPath() == "" {
		stmt := conn.Prep("SELECT url, date, title FROM posts;")

		posts := ""

		for {
			if hasRow, err := stmt.Step(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Error trying to query database for posts! " + err.Error()))
				return
			} else if !hasRow {
				break
			}

			url := stmt.GetText("url")
			date := stmt.GetText("date")
			title := stmt.GetText("title")

			posts += fmt.Sprintf("<i>%s</i> <a href=\"/blog/%s\">%s</a><br/>", date, url, title)
		}

		template, err := f.ReadFile("blog.html")

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error trying to load blog template file! %s", err.Error())))
			log.Fatalf("Error trying to load blog template file! %s\n", err.Error())
		}

		finalData := strings.Replace(string(template), "$$$TITLE$$$", "Joe's Blog Posts", 1)
		finalData = strings.Replace(finalData, "$$$BODY$$$", posts, 1)
		finalData = strings.Replace(finalData, "$$$DATE$$$", "", 1)

		w.Write([]byte(finalData))
		return
	}

	stmt := conn.Prep("SELECT url, date, title, body FROM posts WHERE url = $url;")
	stmt.SetText("$url", r.URL.EscapedPath())
	log.Printf("Path of request: %s\n", r.URL.EscapedPath())

	defer stmt.Reset()

	for {
		if hasRow, err := stmt.Step(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error trying to query database for post! " + err.Error()))
			return
		} else if !hasRow {
			break
		}

		date := stmt.GetText("date")
		title := stmt.GetText("title")
		body := stmt.GetText("body")

		template, err := f.ReadFile("blog.html")

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error trying to load blog template file! %s", err.Error())))
			log.Fatalf("Error trying to load blog template file! %s\n", err.Error())
		}

		finalData := strings.Replace(string(template), "$$$TITLE$$$", title, 1)
		finalData = strings.Replace(finalData, "$$$BODY$$$", body, 1)
		finalData = strings.Replace(finalData, "$$$DATE$$$", date, 1)

		w.Write([]byte(finalData))
		return
	}

	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("No matching post!"))
}

func assertSchemas() {
	conn := dbpool.Get(nil)

	if conn == nil {
		return
	}

	defer dbpool.Put(conn)

	stmt, err := conn.Prepare("SELECT id, url, date, title, body FROM posts;")
	if stmt == nil || err != nil {
		stmtdrop, errdrop := conn.Prepare("DROP TABLE posts;")
		if stmtdrop != nil || errdrop == nil {
			stmtdrop.Step()
			stmtdrop.Finalize()
		}

		stmtcreate := conn.Prep("CREATE TABLE posts (id INTEGER PRIMARY KEY, url, date, title, body);")
		stmtcreate.Step()
		stmtcreate.Finalize()
	} else {
		stmt.Finalize()
	}

}
