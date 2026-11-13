package services

import (
    "bytes"
    "fmt"
    "ghost2gemini/models/config"
    "net/url"
    "regexp"
    "strings"

    "github.com/LukeEmmet/html2gemini"
    "golang.org/x/net/html"
    "golang.org/x/net/html/atom"
)

const codeBlockPlaceholder = "GHOSTGEMLOGCODEBLOCK%dEND"

// html2gemini puts several spaces between the address and the name of a link
var linkSpacingPattern = regexp.MustCompile(`(?m)^=> (\S+) {2,}`)

func htmlToGemini(input string) (string, error) {
    doc, err := html.Parse(strings.NewReader(input))
    if err != nil {
        return "", err
    }

    codeBlocks := prepareGhostHtml(doc)

    var buf bytes.Buffer
    if err := html.Render(&buf, doc); err != nil {
        return "", err
    }
    input = buf.String()
    input = strings.ReplaceAll(input, "</p>", "</p><br><br>")
    input = strings.ReplaceAll(input, "</div>", "</div><br><br>")

    options := html2gemini.NewOptions()
    options.CitationStart = 4
    options.LinkEmitFrequency = 1
    options.CitationMarkers = true
    options.NumberedLinks = true
    options.EmitImagesAsLinks = true
    options.ImageMarkerPrefix = "🖼️"

    options.PrettyTables = false
    options.PrettyTablesOptions.HeaderLine = true
    options.PrettyTablesOptions.RowLine = true
    options.PrettyTablesOptions.CenterSeparator = " "
    options.PrettyTablesOptions.ColumnSeparator = " "
    options.PrettyTablesOptions.RowSeparator = " "

    ctx := html2gemini.NewTraverseContext(*options)

    text, err := html2gemini.FromString(input, *ctx)
    if err != nil {
        return "", err
    }

    text = strings.ReplaceAll(text, "=> "+config.BlogHost, "=> ")
    text = linkSpacingPattern.ReplaceAllString(text, "=> $1 ")

    for i, codeBlock := range codeBlocks {
        text = strings.Replace(text, fmt.Sprintf(codeBlockPlaceholder, i), codeBlock, 1)
    }

    return text, nil
}

// prepareGhostHtml rewrites the cards of the Ghost editor that html2gemini can't
// display into plain html. Code blocks are cut out and returned separately,
// because html2gemini strips their indentation and blank lines.
func prepareGhostHtml(doc *html.Node) []string {
    var nodes []*html.Node
    var collect func(*html.Node)
    collect = func(n *html.Node) {
        if n.Type == html.ElementNode {
            nodes = append(nodes, n)
        }
        for c := n.FirstChild; c != nil; c = c.NextSibling {
            collect(c)
        }
    }
    collect(doc)

    codeBlocks := make([]string, 0)
    for _, n := range nodes {
        if isDetached(n, doc) {
            continue
        }

        switch {
        case n.Data == "s" || n.Data == "strike":
            if config.TextStrikeTag != "" {
                n.Data = config.TextStrikeTag
                n.DataAtom = 0
            }
        case n.Data == "pre":
            placeholder := fmt.Sprintf(codeBlockPlaceholder, len(codeBlocks))
            codeBlocks = append(codeBlocks, codeBlockToGemini(n))
            replaceNode(n, newParagraph(placeholder))
        case n.Data == "iframe":
            src := embedToLink(getAttr(n, "src"))
            if src == "" {
                removeNode(n)
                continue
            }
            title := getAttr(n, "title")
            if title == "" {
                title = src
            }
            replaceNode(n, newLinkParagraph(src, "▶️ "+title))
        case hasClass(n, "kg-bookmark-card"):
            link := findNode(n, func(c *html.Node) bool { return c.Data == "a" })
            if link == nil {
                continue
            }
            title := textOfClass(n, "kg-bookmark-title")
            if title == "" {
                title = getAttr(link, "href")
            }
            if author := textOfClass(n, "kg-bookmark-author"); author != "" && author != title {
                title += " — " + author
            }
            replacement := []*html.Node{newLinkParagraph(getAttr(link, "href"), title)}
            if description := textOfClass(n, "kg-bookmark-description"); description != "" {
                replacement = append(replacement, newParagraph(description))
            }
            if caption := findNode(n, func(c *html.Node) bool { return c.Data == "figcaption" }); caption != nil {
                removeNode(caption)
                replacement = append(replacement, caption)
            }
            replaceNode(n, replacement...)
        case hasClass(n, "kg-video-card"), hasClass(n, "kg-audio-card"):
            media := findNode(n, func(c *html.Node) bool { return c.Data == "video" || c.Data == "audio" })
            if media == nil || getAttr(media, "src") == "" {
                continue
            }
            title := "🎞️ Video"
            if media.Data == "audio" {
                title = "🔊 " + textOfClass(n, "kg-audio-title")
            }
            replacement := []*html.Node{newLinkParagraph(getAttr(media, "src"), title)}
            if caption := findNode(n, func(c *html.Node) bool { return c.Data == "figcaption" }); caption != nil {
                removeNode(caption)
                replacement = append(replacement, caption)
            }
            replaceNode(n, replacement...)
        case hasClass(n, "kg-file-card"):
            link := findNode(n, func(c *html.Node) bool { return c.Data == "a" })
            if link == nil {
                continue
            }
            title := textOfClass(n, "kg-file-card-title")
            if title == "" {
                title = textOfClass(n, "kg-file-card-filename")
            }
            replacement := []*html.Node{newLinkParagraph(getAttr(link, "href"), "📎 "+title)}
            if caption := textOfClass(n, "kg-file-card-caption"); caption != "" {
                replacement = append(replacement, newParagraph(caption))
            }
            replaceNode(n, replacement...)
        case hasClass(n, "kg-signup-card"):
            // member signup forms make no sense outside of the web
            removeNode(n)
        }
    }

    return codeBlocks
}

func codeBlockToGemini(pre *html.Node) string {
    language := ""
    if code := findNode(pre, func(c *html.Node) bool { return c.Data == "code" }); code != nil {
        for _, class := range strings.Fields(getAttr(code, "class")) {
            if strings.HasPrefix(class, "language-") {
                language = strings.TrimPrefix(class, "language-")
            }
        }
    }

    lines := strings.Split(strings.Trim(strings.ReplaceAll(rawText(pre), "\r\n", "\n"), "\n"), "\n")
    for i, line := range lines {
        // keep a line of the code from closing the preformatted block
        if strings.HasPrefix(line, "```") {
            lines[i] = " " + line
        }
    }

    return "```" + language + "\n" + strings.Join(lines, "\n") + "\n```"
}

// embedToLink turns the address of an embedded player into the address of its page
func embedToLink(src string) string {
    if strings.HasPrefix(src, "//") {
        src = "https:" + src
    }
    parsed, err := url.Parse(src)
    if err != nil || parsed.Host == "" {
        return ""
    }

    host := strings.TrimPrefix(parsed.Host, "www.")
    if (host == "youtube.com" || host == "youtube-nocookie.com") && strings.HasPrefix(parsed.Path, "/embed/") {
        return "https://www.youtube.com/watch?v=" + strings.TrimPrefix(parsed.Path, "/embed/")
    }
    if host == "player.vimeo.com" && strings.HasPrefix(parsed.Path, "/video/") {
        return "https://vimeo.com/" + strings.TrimPrefix(parsed.Path, "/video/")
    }

    return src
}

func getAttr(n *html.Node, name string) string {
    for _, attr := range n.Attr {
        if attr.Key == name {
            return attr.Val
        }
    }
    return ""
}

func hasClass(n *html.Node, class string) bool {
    for _, c := range strings.Fields(getAttr(n, "class")) {
        if c == class {
            return true
        }
    }
    return false
}

func findNode(n *html.Node, match func(*html.Node) bool) *html.Node {
    for c := n.FirstChild; c != nil; c = c.NextSibling {
        if c.Type == html.ElementNode && match(c) {
            return c
        }
        if found := findNode(c, match); found != nil {
            return found
        }
    }
    return nil
}

func rawText(n *html.Node) string {
    if n.Type == html.TextNode {
        return n.Data
    }
    var text string
    for c := n.FirstChild; c != nil; c = c.NextSibling {
        text += rawText(c)
    }
    return text
}

func textOfClass(n *html.Node, class string) string {
    found := findNode(n, func(c *html.Node) bool { return hasClass(c, class) })
    if found == nil {
        return ""
    }
    return strings.Join(strings.Fields(rawText(found)), " ")
}

func isDetached(n *html.Node, root *html.Node) bool {
    for n != nil {
        if n == root {
            return false
        }
        n = n.Parent
    }
    return true
}

func removeNode(n *html.Node) {
    if n.Parent != nil {
        n.Parent.RemoveChild(n)
    }
}

func replaceNode(old *html.Node, replacement ...*html.Node) {
    if old.Parent == nil {
        return
    }
    for _, n := range replacement {
        old.Parent.InsertBefore(n, old)
    }
    old.Parent.RemoveChild(old)
}

func newParagraph(text string) *html.Node {
    p := &html.Node{Type: html.ElementNode, Data: "p", DataAtom: atom.P}
    p.AppendChild(&html.Node{Type: html.TextNode, Data: text})
    return p
}

func newLinkParagraph(href string, text string) *html.Node {
    a := &html.Node{
        Type:     html.ElementNode,
        Data:     "a",
        DataAtom: atom.A,
        Attr:     []html.Attribute{{Key: "href", Val: href}},
    }
    a.AppendChild(&html.Node{Type: html.TextNode, Data: text})
    p := &html.Node{Type: html.ElementNode, Data: "p", DataAtom: atom.P}
    p.AppendChild(a)
    return p
}
