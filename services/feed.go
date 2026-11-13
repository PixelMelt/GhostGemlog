package services

import (
    "encoding/json"
    "fmt"
    "ghost2gemini/models"
    "ghost2gemini/models/config"
    "os"
    "path/filepath"
)

const jsonFilePath = "data/blog/posts.json"

func SaveFeed() error {
    rssFeed, err := GetGhostPostList()
    if err != nil {
        return err
    }

    // the whole list of posts is requested from the blog,
    // so posts that were deleted or unpublished disappear from the capsule too
    rssFeed.Channel.SortItemsByPubDate()

    jsonData, err := json.MarshalIndent(rssFeed, "", "  ")
    if err != nil {
        return err
    }

    err = os.MkdirAll(filepath.Dir(jsonFilePath), 0755)
    if err != nil {
        return err
    }

    tmpFilePath := jsonFilePath + ".tmp"
    err = os.WriteFile(tmpFilePath, jsonData, 0644)
    if err != nil {
        return err
    }

    return os.Rename(tmpFilePath, jsonFilePath)
}

func GetFeedAsGemini() (string, error) {
    rssFeed, err := readRssFromJson()
    if err != nil {
        return "", err
    }

    geminiContent := fmt.Sprintf(
        "# %s\n\n## %s\n\n%s\n\n",
        rssFeed.Channel.Title,
        rssFeed.Channel.Description,
        config.TextDescription,
    )
    for _, item := range rssFeed.Channel.Items {
        geminiContent += fmt.Sprintf(
            "### %s\n\n%s\n\n=> /%s/ %s | %s\n\n",
            item.Title,
            item.Description,
            item.Link,
            item.PubDate,
            config.TextRead,
        )
    }

    return geminiContent, nil
}

func readRssFromJson() (*models.Feed, error) {
    byteValue, err := os.ReadFile(jsonFilePath)
    if err != nil {
        return nil, err
    }

    var rssFeed models.Feed
    err = json.Unmarshal(byteValue, &rssFeed)
    if err != nil {
        return nil, err
    }

    return &rssFeed, nil
}
