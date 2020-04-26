# reddit-post-crawler-video-generator

Get popular GIFs and videos on the Internet and aggregate them into a video.

### How It Works

This app works with Chrome (headless), ffmpeg, and youtube-dl wrapped in a Docker image to ingest, process, and produce content from your favorite Reddit subs.


### Getting Started

```
# Build image
docker build -t rpcvg .

# Run container with single ingestion
docker run --rm --name rpcvg \
    -v /opt/rpcvg/cache:/cache \
    -v /opt/rpcvg/sources:/data \
    -v /opt/rpcvg/output:/output \
    rpcvg:latest \
    https://www.reddit.com/r/BetterEveryLoop/top/?t=year

# Run container with multiple ingestions
docker ... rpcvg:latest \
    https://www.reddit.com/r/BetterEveryLoop/top/?t=year \
    https://www.reddit.com/r/gifs/top/?t=year \
    https://www.reddit.com/r/funny/top/?t=year
```

#### Windows

Here are some example paths you can use:

```
-v C:\Users\YOURUSER\Desktop\rpcvg\cache:/cache
-v C:\Users\YOURUSER\Desktop\rpcvg\source:/data
-v C:\Users\YOURUSER\Desktop\rpcvg\output:/output
```
