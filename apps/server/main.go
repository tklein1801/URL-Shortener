package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var ctx = context.Background()
var authCode string

func main() {
	missingEnvVars := CheckForEnvironmentVariables([]string{"REDIS_HOST", "REDIS_PW", "REDIS_DB", "CODE"})
	if len(missingEnvVars) > 0 {
		log.Fatalf("Missing environment variables: %v", missingEnvVars)
	}

	authCode = os.Getenv("CODE")
	redisDbNum, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		log.Fatalf("Error parsing REDIS_DB: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST"),
		Password: os.Getenv("REDIS_PW"),
		DB:       redisDbNum,
	})

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/health"))

	r.Get("/list", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != authCode {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		keys, _, err := rdb.Scan(ctx, 0, "*", 0).Result()
		if err != nil {
			panic(err)
		}

		data := make(map[string]string)
		for _, key := range keys {
			value, err := rdb.Get(ctx, key).Result()
			if err != nil {
				panic(err)
			}
			data[key] = value
		}

		jsonData, err := json.Marshal(map[string]interface{}{"data": data})
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(jsonData)
		if err != nil {
			panic(err)
		}
	})

	r.Post("/shorten", func(w http.ResponseWriter, r *http.Request) {
		originalUrl := r.FormValue("url")
		if originalUrl == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		shortUrl := generateRandomString(8)
		err := rdb.Set(ctx, shortUrl, originalUrl, 0).Err()
		if err != nil {
			panic(err)
		}

		jsonData, err := json.Marshal(map[string]string{"shortUrl": shortUrl})
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(jsonData)
		if err != nil {
			panic(err)
		}
	})

	r.Get("/r/{shortUrl}", func(w http.ResponseWriter, r *http.Request) {
		shortUrl := chi.URLParam(r, "shortUrl")
		if shortUrl == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		val, err := rdb.Get(ctx, shortUrl).Result()
		if err != nil {
			println(err.Error())
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		if val == "" {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		http.Redirect(w, r, val, http.StatusTemporaryRedirect)
	})

	r.Delete("/d/{shortUrl}", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != authCode {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		shortUrl := chi.URLParam(r, "shortUrl")
		if shortUrl == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		err := rdb.Del(ctx, shortUrl).Err()
		if err != nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		jsonData, err := json.Marshal(map[string]string{"shortUrl": shortUrl})
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(jsonData)
		if err != nil {
			panic(err)
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	_, err = strconv.Atoi(port)
	if err != nil {
		log.Fatalf("Invalid port number: %v", err)
	}

	log.Printf("Starting server under http://localhost:%s", port)
	err = http.ListenAndServe(fmt.Sprintf(":%s", port), r)
	if err != nil {
		log.Fatalf("Server failed to start on port %s: %v", port, err)
	}
}

func FileExists(path string) bool {
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return !info.IsDir()
}

func CheckForEnvironmentVariables(variables []string) []string {
	if FileExists(".env") {
		err := godotenv.Load(".env")
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	} else {
		log.Println("No .env file found")
	}

	var missingEnvVars []string
	for _, variable := range variables {
		if os.Getenv(variable) == "" {
			missingEnvVars = append(missingEnvVars, variable)
		}
	}
	return missingEnvVars
}

func generateRandomString(length int) string {
	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}
	randomString := base64.URLEncoding.EncodeToString(randomBytes)
	return randomString[:length]
}
