package main

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/harsh082ip/ZapTun/internal/server/github"
)

var oauth github.Authenticator

//go:embed static/config.json
var config string

//go:embed static/index.html
var html string

//go:embed static/token.html
var tokenHtml string

func main() {
	clientId := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	if clientId == "" || clientSecret == "" {
		log.Fatalf("missing github client id/secret")
	}
	oauth = github.New(clientId, clientSecret)

	log.Println("Starting server on http://localhost:8080")

	http.HandleFunc("/", serveStaticContent([]byte(html), "text/html"))
	http.HandleFunc("/config.json", serveStaticContent([]byte(config), "application/json"))
	http.HandleFunc("/auth", authHandler)
	http.HandleFunc("/auth-callback", authCallback)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func serveStaticContent(content []byte, contentType string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Write(content)
	}
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, oauth.GetOAuthUrl(), http.StatusFound)
}

func authCallback(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil || r.FormValue("code") == "" {
		http.Redirect(w, r, "/auth", http.StatusTemporaryRedirect)
		return
	}
	token, err := oauth.ExchangeCodeForToken(r.FormValue("code"))
	if err != nil || token == "" {
		fmt.Printf("error obtaining token: %s\n", err)
		http.Redirect(w, r, "/auth", http.StatusTemporaryRedirect)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	// TO:
	w.Write([]byte(strings.Replace(tokenHtml, "##TOKEN##", token, 1)))
}
