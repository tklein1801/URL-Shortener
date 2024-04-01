# Golang URL shortener

## Getting started

> [!NOTE]
> In order to run this application, you need to have a working Redis instance running.

1. Clone the repository
2. Set all required environment variables (as defined in `.env.example`)
3. Start your application
    ```bash
    go run main.go
    ```
4. Open your browser and go to `http://localhost:3000/`

## API

### Create a new short URL

> POST /shorten

```bash
curl -X POST http://localhost:3000/shorten \
    -d 'url=https://www.google.com'
```

### Get a short URL

> GET /r/:shortUrl
 
```bash
curl http://localhost:3000/r/:shortUrl
```

### Delete a short URL

> DELETE /d/:shortUrl

```bash
curl -X DELETE http://localhost:3000/d/:shortUrl
```