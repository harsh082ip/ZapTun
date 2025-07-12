package main

import (
	"fmt"
	"log"
	"net/http"
)

const APP_PORT = ":8080"

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `
			<!DOCTYPE html>
			<html lang="en">
			<head>
				<meta charset="UTF-8">
				<title>You Did It!</title>
				<style>
					body {
						background: linear-gradient(135deg, #ff9a9e 0%, #fad0c4 100%);
						font-family: 'Comic Sans MS', cursive, sans-serif;
						color: #222;
						text-align: center;
						padding: 50px;
						overflow-x: hidden;
					}
					h1 {
						font-size: 3em;
						color: #fff;
						text-shadow: 2px 2px 5px #000;
						margin-bottom: 20px;
					}
					p {
						font-size: 1.5em;
						margin-bottom: 40px;
					}
					img {
						width: 400px;
						max-width: 80%%;
						border-radius: 15px;
						box-shadow: 0 0 20px rgba(0, 0, 0, 0.6);
					}
					@keyframes confetti {
						0%% {transform: translateY(0);}
						100%% {transform: translateY(100vh);}
					}
					.confetti {
						position: absolute;
						width: 10px;
						height: 10px;
						background: #fff;
						opacity: 0.7;
						border-radius: 50%%;
						animation: confetti 3s linear infinite;
					}
				</style>
			</head>
			<body>
				<h1>🎉 Wow! You actually got in! 🎉</h1>
				<p>Here’s your well-deserved *reward* (or is it a trap? 😏)</p>
				<img src="https://media.giphy.com/media/Vuw9m5wXviFIQ/giphy.gif" alt="Rickroll">

				<script>
					// Generate falling confetti
					for (let i = 0; i < 50; i++) {
						let confetti = document.createElement('div');
						confetti.classList.add('confetti');
						confetti.style.left = Math.random() * 100 + 'vw';
						confetti.style.top = Math.random() * -100 + 'vh';
						confetti.style.backgroundColor = '#' + Math.floor(Math.random()*16777215).toString(16);
						confetti.style.animationDuration = (Math.random() * 3 + 2) + 's';
						document.body.appendChild(confetti);
					}
				</script>
			</body>
			</html>
		`)
	})

	log.Println("Starting server on port", APP_PORT)
	if err := http.ListenAndServe(APP_PORT, nil); err != nil {
		log.Fatal("Failed to start server on port", APP_PORT)
	}
}
