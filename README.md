# reddit-post-crawler-video-generator

Get popular GIFs and videos on the Internet and aggregate them into a video.

### How It Works

This app works with Chrome (headless), ffmpeg, and youtube-dl wrapped in a Docker image to ingest, process, and produce content from your favorite Reddit subs.

### Getting Started

```
docker build -t rpcvg .
docker run rpcvg
```
