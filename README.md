# Golang URL shortener

- [Golang URL shortener](#golang-url-shortener)
  - [Server](#server)
    - [Getting started](#getting-started)
    - [API](#api)
      - [Get the list of short URLs](#get-the-list-of-short-urls)
      - [Create a new short URL](#create-a-new-short-url)
      - [Get a short URL](#get-a-short-url)
      - [Delete a short URL](#delete-a-short-url)
  - [CLI](#cli)

## Server

### Getting started

> [!NOTE]
> In order to run this application, you need to have a working Redis instance running.

1. Clone the repository
2. Set all required environment variables (as defined in `.env.example`)
3. Start your application
   ```bash
   go run main.go
   ```
4. Open your browser and go to `http://localhost:3000/`

### API

#### Get the list of short URLs

> GET /list

```bash
curl http://localhost:3000/list?code=<AUTH_CODE>
```

#### Create a new short URL

> POST /shorten

```bash
curl -X POST http://localhost:3000/shorten \
    -d 'url=https://www.google.com'
```

#### Get a short URL

> GET /r/:shortUrl

```bash
curl http://localhost:3000/r/:shortUrl?code=<AUTH_CODE>
```

#### Delete a short URL

> DELETE /d/:shortUrl

```bash
curl -X DELETE http://localhost:3000/d/:shortUrl
```

## CLI
