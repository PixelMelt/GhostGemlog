package config

import (
    "os"
    "strconv"
    "strings"

    "github.com/joho/godotenv"
)

var (
    BlogHost           string
    BlogApiHost        string
    BlogKey            string
    GeminiHost         string
    GeminiPort         string
    FeedUpdateInterval int
    TextDescription    string
    TextFooter         string
    TextRead           string
    TextStrikeTag      string
)

func LoadConfig() {
    godotenv.Load()

    BlogHost = strings.TrimSuffix(os.Getenv("BLOG_HOST"), "/")
    BlogApiHost = strings.TrimSuffix(os.Getenv("BLOG_API_HOST"), "/")
    if BlogApiHost == "" {
        BlogApiHost = BlogHost
    }
    BlogKey = os.Getenv("BLOG_KEY")
    GeminiHost = os.Getenv("GEMINI_HOST")
    GeminiPort = os.Getenv("GEMINI_PORT")
    if GeminiPort == "" {
        GeminiPort = "1965"
    }
    FeedUpdateInterval = 15
    if interval, err := strconv.Atoi(os.Getenv("FEED_UPDATE_INTERVAL")); err == nil {
        FeedUpdateInterval = interval
    }
    TextDescription = os.Getenv("TEXT_DESCRIPTION")
    TextFooter = os.Getenv("TEXT_FOOTER")
    TextRead = os.Getenv("TEXT_READ")
    TextStrikeTag = os.Getenv("TEXT_STRIKE_TAG")
}
