package main

import (
    "context"
    "crypto/tls"
    "ghost2gemini/handlers"
    "ghost2gemini/models/config"
    "ghost2gemini/services"
    "github.com/ninedraft/gemax/gemax"
    "log"
    "time"
)

func main() {
    config.LoadConfig()

    server := &gemax.Server{
        Addr:    config.GeminiHost + ":" + config.GeminiPort,
        Handler: handlers.BlogHandler,
    }

    cert, errCert := tls.LoadX509KeyPair("data/cert/cert.pem", "data/cert/key.pem")
    if errCert != nil {
        log.Fatal(errCert)
    }

    cfg := &tls.Config{
        MinVersion:   tls.VersionTLS12,
        Certificates: []tls.Certificate{cert},
    }

    if config.FeedUpdateInterval > 0 {
        go updateFeed(time.Duration(config.FeedUpdateInterval) * time.Minute)
    }

    log.Println("Starting server listening on " + config.GeminiHost + ":" + config.GeminiPort)
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    err := server.ListenAndServe(ctx, cfg)
    if err != nil {
        log.Printf("test server: Serve: %v", err)
    }
}

// updateFeed keeps the list of posts on the main page up to date
func updateFeed(interval time.Duration) {
    for {
        err := services.SaveFeed()
        if err != nil {
            // the blog may be still starting, don't wait for the whole interval
            log.Println("Can't update the list of posts:", err)
            time.Sleep(30 * time.Second)
            continue
        }
        time.Sleep(interval)
    }
}
