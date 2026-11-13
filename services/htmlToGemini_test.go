package services

import (
    "ghost2gemini/models/config"
    "strings"
    "testing"
)

// the fixtures are cards as they are rendered by the Content API of Ghost 6

func TestBookmarkCard(t *testing.T) {
    input := `<figure class="kg-card kg-bookmark-card"><a class="kg-bookmark-container" href="https://github.com/example?ref=blog.example.com"><div class="kg-bookmark-content"><div class="kg-bookmark-title">example - Overview</div><div class="kg-bookmark-description">Example has 50 repositories available.</div><div class="kg-bookmark-metadata"><img class="kg-bookmark-icon" src="https://blog.example.com/content/images/icon/favicon.svg" alt=""><span class="kg-bookmark-author">GitHub</span></div></div><div class="kg-bookmark-thumbnail"><img src="https://blog.example.com/content/images/thumbnail/1138" alt="" onerror="this.style.display = 'none'"></div></a></figure>`

    text, err := htmlToGemini(input)
    if err != nil {
        t.Fatal(err)
    }

    if !strings.Contains(text, "=> https://github.com/example?ref=blog.example.com example - Overview — GitHub") {
        t.Errorf("bookmark is not a link line:\n%s", text)
    }
    if !strings.Contains(text, "Example has 50 repositories available.") {
        t.Errorf("bookmark description is lost:\n%s", text)
    }
    if strings.Contains(text, "🖼️") {
        t.Errorf("bookmark icons are displayed:\n%s", text)
    }
}

func TestEmbedCard(t *testing.T) {
    input := `<figure class="kg-card kg-embed-card kg-card-hascaption"><iframe width="200" height="113" src="https://www.youtube.com/embed/k1xBxpEZ8iQ?feature=oembed" frameborder="0" allowfullscreen="" title="Some video"></iframe><figcaption><p><i><em class="italic" style="white-space: pre-wrap;">Caption of the video</em></i></p></figcaption></figure>`

    text, err := htmlToGemini(input)
    if err != nil {
        t.Fatal(err)
    }

    if !strings.Contains(text, "=> https://www.youtube.com/watch?v=k1xBxpEZ8iQ ▶️ Some video") {
        t.Errorf("embed is not a link line:\n%s", text)
    }
    if !strings.Contains(text, "Caption of the video") {
        t.Errorf("caption is lost:\n%s", text)
    }
}

func TestCodeCard(t *testing.T) {
    input := "<p>Before</p><figure class=\"kg-card kg-code-card\"><pre><code class=\"language-js\">const err = new Error();\nObject.defineProperty(err, 'stack', {\n  get() {\n    console.log(\"a &lt; b\")\n\n\n  }\n});\n</code></pre></figure><p>After</p>"
    expected := "```js\nconst err = new Error();\nObject.defineProperty(err, 'stack', {\n  get() {\n    console.log(\"a < b\")\n\n\n  }\n});\n```"

    text, err := htmlToGemini(input)
    if err != nil {
        t.Fatal(err)
    }

    if !strings.Contains(text, expected) {
        t.Errorf("code block is changed:\n%s", text)
    }
    if !strings.HasPrefix(text, "Before") || !strings.HasSuffix(text, "After") {
        t.Errorf("text around the code block is lost:\n%s", text)
    }
}

func TestLinksToBlogAreRelative(t *testing.T) {
    config.BlogHost = "https://blog.example.com"
    defer func() { config.BlogHost = "" }()

    input := `<figure class="kg-card kg-image-card"><img src="https://blog.example.com/content/images/2026/07/image-3.png" class="kg-image" alt="" loading="lazy"></figure>`

    text, err := htmlToGemini(input)
    if err != nil {
        t.Fatal(err)
    }

    if !strings.Contains(text, "=> /content/images/2026/07/image-3.png") {
        t.Errorf("image link is not relative:\n%s", text)
    }
}

func TestSignupCardIsRemoved(t *testing.T) {
    input := `<p>Text</p><div class="kg-card kg-signup-card"><h2 class="kg-signup-card-heading">Sign up</h2><form class="kg-signup-card-form" data-members-form="signup"><input class="kg-signup-card-input" type="email"></form></div>`

    text, err := htmlToGemini(input)
    if err != nil {
        t.Fatal(err)
    }

    if strings.TrimSpace(text) != "Text" {
        t.Errorf("signup card is displayed:\n%s", text)
    }
}
