package services

import (
    "fmt"
    "net/http"
)

// ProxyFile requests a file from the blog, the caller has to close the body of the response
func ProxyFile(reqUrl string) (*http.Response, error) {
    req, err := newBlogRequest(reqUrl)
    if err != nil {
        return nil, err
    }

    resp, err := httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    if resp.StatusCode >= 400 {
        resp.Body.Close()
        return nil, fmt.Errorf("file %s returned status %d", reqUrl, resp.StatusCode)
    }

    return resp, nil
}
