# reddit-post-crawler-video-generator

Get popular GIFs and videos on the Internet and aggregate them into a video.

### How It Works

This app works with Chrome (headless), ffmpeg, and youtube-dl wrapped in a Docker image to ingest, process, and produce content from your favorite Reddit subs.

### Getting Started

```
# Build image
docker build -t rpcvg .

# Run container
## Linux / MacOS ##
docker run -v /opt/rpcvg/sources:/data -v /opt/rpcvg/output:/output rpcvg:latest

## Windows ##
docker run -v C:\Users\YOURUSER\Desktop\rpcvg\source:/data -v C:\Users\YOURUSER\Desktop\rpcvg\output:/output rpcvg:latest
```
