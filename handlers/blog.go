package handlers

import (
    "context"
    "ghost2gemini/models/config"
    "ghost2gemini/services"
    "github.com/ninedraft/gemax/gemax"
    "github.com/ninedraft/gemax/gemax/status"
    "io"
    "log"
    "net/http"
    "os"
    "path"
    "regexp"
    "strings"
    "time"
)

var slugPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func BlogHandler(_ context.Context, rw gemax.ResponseWriter, req gemax.IncomingRequest) {
    err := logRequest(req)
    if err != nil {
        log.Println(err)
    }

    urlPath := path.Clean("/" + req.URL().Path)
    urlPath = strings.TrimPrefix(urlPath, "/")
    if urlPath == "" {
        handleIndex(rw)
        return
    }

    if urlPath == "updateBlogPosts" {
        handleUpdateFeed(rw)
        return
    }

    fileContent, isMedia, err := services.GetFile(urlPath)
    if err != nil {
        rw.WriteStatus(status.ServerUnavailable, "Can't display contents of: "+req.URL().Path)
        log.Println(err)
        return
    }

    if fileContent != nil {
        handleLocalFile(isMedia, fileContent, rw)
        return
    }

    if isFileURL(urlPath) {
        handleFileProxy(config.BlogApiHost+"/"+urlPath, rw, req)
        return
    }

    handlePostProxy(urlPath, rw, req)
}

func handleLocalFile(isMedia bool, fileContent []byte, rw gemax.ResponseWriter) {
    if isMedia {
        contentType := http.DetectContentType(fileContent)
        rw.WriteStatus(status.Success, contentType)
    }
    rw.Write(fileContent)
    if !isMedia {
        rw.Write([]byte(config.TextFooter))
    }
}

func logRequest(request gemax.IncomingRequest) error {
    logFilename := "logs/access.log"
    file, err := os.OpenFile(logFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer file.Close()

    logger := log.New(file, "", log.LstdFlags)

    currentTime := time.Now().Format("2006-01-02 15:04:05")
    logger.Printf(
        "%s | IP %s | %s \n",
        currentTime,
        request.RemoteAddr(),
        request.URL(),
    )

    return nil
}

func handlePostProxy(urlPath string, rw gemax.ResponseWriter, req gemax.IncomingRequest) {
    // posts are requested by their slug, which is the last part of the path
    // even when the blog uses dated or prefixed permalinks
    slug := path.Base(urlPath)
    if !slugPattern.MatchString(slug) {
        rw.WriteStatus(status.NotFound, "Not found: "+req.URL().Path)
        return
    }

    postText, err := services.GetGhostPost(slug)
    if err == nil {
        io.WriteString(rw, postText+config.TextFooter)
        return
    }

    pageText, err := services.GetGhostPage(slug)
    if err == nil {
        io.WriteString(rw, pageText+config.TextFooter)
        return
    }

    rw.WriteStatus(status.NotFound, "Not found: "+req.URL().Path)
}

func handleFileProxy(reqUrl string, rw gemax.ResponseWriter, req gemax.IncomingRequest) {
    resp, err := services.ProxyFile(reqUrl)
    if err != nil {
        log.Println(err)
        rw.WriteStatus(status.NotFound, "Not found: "+req.URL().Path)
        return
    }
    defer resp.Body.Close()

    contentType := resp.Header.Get("Content-Type")
    if contentType == "" {
        contentType = "application/octet-stream"
    }
    rw.WriteStatus(status.Success, contentType)
    _, err = io.Copy(rw, resp.Body)
    if err != nil {
        log.Println(err)
    }
}

func handleUpdateFeed(rw gemax.ResponseWriter) {
    err := services.SaveFeed()
    if err != nil {
        log.Println(err)
    }
    io.WriteString(rw, "Updated")
}

func handleIndex(rw gemax.ResponseWriter) {
    geminiContent, _ := services.GetFeedAsGemini()
    io.WriteString(rw, geminiContent+config.TextFooter)
}

// isFileURL reports whether the path is a file stored by Ghost: images, media and
// files of the editor cards are kept in the content directory
func isFileURL(urlPath string) bool {
    prefixes := []string{"content/images/", "content/media/", "content/files/"}
    for _, prefix := range prefixes {
        if strings.HasPrefix(urlPath, prefix) {
            return true
        }
    }

    suffixes := []string{".jpg", ".jpeg", ".webp", ".png", ".gif", ".svg", ".avif"}
    lowerUrlPath := strings.ToLower(urlPath)
    for _, suffix := range suffixes {
        if strings.HasSuffix(lowerUrlPath, suffix) {
            return true
        }
    }

    return false
}
