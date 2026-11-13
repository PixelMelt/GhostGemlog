package services

import (
    "encoding/json"
    "errors"
    "fmt"
    "ghost2gemini/models"
    "ghost2gemini/models/config"
    "io"
    "net/http"
    "net/url"
    "strconv"
    "strings"
    "time"
)

// Ghost 6 no longer accepts limit=all and caps page size at 100
const ghostPageLimit = 100
const ghostApiVersion = "v6.0"

var httpClient = &http.Client{Timeout: 30 * time.Second}

func GetGhostPost(slug string) (string, error) {
    rawPost, err := getFromApi("posts/slug/"+slug, nil)
    if err != nil {
        return "", err
    }

    var posts models.GhostPosts
    err = json.Unmarshal(rawPost, &posts)
    if err != nil {
        return "", err
    }

    return parseGhostItem(posts.Posts)
}

func GetGhostPage(slug string) (string, error) {
    rawPage, err := getFromApi("pages/slug/"+slug, nil)
    if err != nil {
        return "", err
    }

    var pages models.GhostPages
    err = json.Unmarshal(rawPage, &pages)
    if err != nil {
        return "", err
    }

    return parseGhostItem(pages.Pages)
}

func GetGhostPostList() (*models.Feed, error) {
    rawSettings, err := getFromApi("settings", nil)
    if err != nil {
        return nil, err
    }

    var settings models.GhostSettings
    err = json.Unmarshal(rawSettings, &settings)
    if err != nil {
        return nil, err
    }

    feed := models.Feed{
        Channel: models.FeedChannel{
            Title:       settings.Content.Title,
            Link:        settings.Content.Url,
            Description: settings.Content.Description,
            Items:       make([]models.FeedItem, 0),
        },
    }

    params := url.Values{}
    params.Set("limit", strconv.Itoa(ghostPageLimit))
    params.Set("filter", "visibility:public")
    params.Set("fields", "id,title,slug,excerpt,custom_excerpt,published_at")

    page := 1
    for page > 0 {
        params.Set("page", strconv.Itoa(page))
        rawPosts, err := getFromApi("posts", params)
        if err != nil {
            return nil, err
        }

        var posts models.GhostPosts
        err = json.Unmarshal(rawPosts, &posts)
        if err != nil {
            return nil, err
        }

        for _, post := range posts.Posts {
            parsedTime, _ := time.Parse(time.RFC3339, post.PubDate)
            pubDate := parsedTime.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")

            item := models.FeedItem{
                Title:       post.Title,
                Link:        post.Slug,
                Description: post.Exceprt,
                PubDate:     pubDate,
                Guid:        post.Id,
            }

            feed.Channel.Items = append(feed.Channel.Items, item)
        }

        // next is null on the last page
        if posts.Meta.Pagination.Next <= page {
            break
        }
        page = posts.Meta.Pagination.Next
    }

    return &feed, nil
}

func parseGhostItem(posts []models.GhostItem) (string, error) {
    if len(posts) == 0 {
        return "", errors.New("not found")
    }
    post := posts[0]

    if post.Visibility != "public" {
        return "", errors.New("forbidden")
    }

    var excerpt string
    if len(post.CustomExceprt) > 0 {
        excerpt = "\n\n### " + post.CustomExceprt
    }

    var imageCaption string
    if len(post.FeatureImageCaption) > 0 {
        imageCaption, _ = htmlToGemini(post.FeatureImageCaption)
        imageCaption = "\n\n" + imageCaption
    }

    var image string
    if len(post.FeatureImage) > 0 {
        image = fmt.Sprintf(
            "\n\n=> %s 🖼️ %s",
            strings.ReplaceAll(post.FeatureImage, config.BlogHost, ""),
            post.Title,
        )
    }

    postText, _ := htmlToGemini(post.Html)

    return "# " + post.Title + excerpt + image + imageCaption + "\n\n" + postText, nil
}

func newBlogRequest(reqUrl string) (*http.Request, error) {
    req, err := http.NewRequest(http.MethodGet, reqUrl, nil)
    if err != nil {
        return nil, err
    }

    // Ghost redirects plain http requests to its public https url
    // unless a proxy says the original request was secure
    if strings.HasPrefix(config.BlogHost, "https://") && strings.HasPrefix(config.BlogApiHost, "http://") {
        req.Header.Set("X-Forwarded-Proto", "https")
    }

    return req, nil
}

func getFromApi(method string, params url.Values) ([]byte, error) {
    if params == nil {
        params = url.Values{}
    }
    params.Set("key", config.BlogKey)

    // Ghost answers requests without a trailing slash with a redirect
    reqUrl := config.BlogApiHost + "/ghost/api/content/" + strings.Trim(method, "/") + "/?" + params.Encode()
    req, err := newBlogRequest(reqUrl)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Accept-Version", ghostApiVersion)

    resp, err := httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    if resp.StatusCode != http.StatusOK {
        var apiErrors models.GhostErrors
        message := resp.Status
        if json.Unmarshal(respBody, &apiErrors) == nil && len(apiErrors.Errors) > 0 {
            message = apiErrors.Errors[0].Message
        }
        return nil, fmt.Errorf("ghost api %s returned status %d: %s", method, resp.StatusCode, message)
    }

    return respBody, nil
}
