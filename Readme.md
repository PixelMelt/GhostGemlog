# GhostGemlog — a proxy for your Ghost blog to the Geminispace
The project is based on [ninedraft/gemax](https://github.com/ninedraft/gemax) and [LukeEmmet/html2gmi](https://github.com/LukeEmmet/html2gmi) libraries

This is a fork of [zojl/GhostGemlog](https://github.com/zojl/GhostGemlog) updated for the Content API of Ghost 6.

### Setting up
Create an `.env` file or use docker with the following environment variables:
- BLOG_HOST — a full url of your blog with https, the same as the `url` in the config of Ghost
- BLOG_API_HOST — optional url to request the blog by, if it differs from BLOG_HOST. Can be an internal address like `http://ghost:2368` when the proxy runs next to Ghost
- BLOG_KEY — content API key from the integrations section of your blog admin panel 
- GEMINI_HOST — host to listen on for the Gemini server, can be 0.0.0.0
- GEMINI_PORT — port of the Gemini server, 1965 by default
- FEED_UPDATE_INTERVAL — minutes between updates of the list of posts on the main page, 15 by default, 0 turns the updates off
- TEXT_DESCRIPTION — custom text for the main page of your capsule
- TEXT_FOOTER — footer text for each page of your capsule
- TEXT_READ — text for post links after their publication date
- TEXT_STRIKE_TAG — name of the ironic tag that will be displayed around the strikethrough text

The list of posts is requested from the blog when the server starts and then every FEED_UPDATE_INTERVAL minutes. It can also be updated at any moment by a request to `gemini://<YOUR_DOMAIN>/updateBlogPosts`.

### Ghost 6
- All requests to the Content API are sent with the `Accept-Version: v6.0` header and a trailing slash
- The list of posts is requested page by page, Ghost 6 doesn't return more than 100 posts at once
- Only public posts are listed
- Bookmark, embed, video, audio and file cards are displayed as links, signup cards are hidden
- Code blocks keep their indentation, blank lines and language
- Files from `/content/images`, `/content/media` and `/content/files` are proxied from the blog

## Example docker-compose.yml
The project is cloned to the src directory.
``` yaml
services:
  app:
    build: src
    env_file: .env
    volumes:
     - ./volume/app/data/blog:/app/data/blog
     - ./volume/app/data/cert:/app/data/cert
     - ./volume/app/logs:/app/logs
    ports:
     - 1965:1965
```

## Dont forget about the certificate
Simply run `openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes -subj '/CN=<YOUR_DOMAIN>'` to generate the certificate with key and put them to your data/cert directory.

## Tests
Run `go test ./...`
