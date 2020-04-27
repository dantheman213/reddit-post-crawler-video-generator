# reddit-post-crawler-video-generator

Get popular GIFs and videos on the Internet and aggregate them into a video.

### How It Works

This app works with Chrome (headless), ffmpeg, and youtube-dl wrapped in a Docker image to ingest, process, and produce content from your favorite Reddit subs.


### Getting Started

#### Get this Docker image from DockerHub

[dantheman213/rpcvg](https://hub.docker.com/repository/docker/dantheman213/rpcvg)

```
docker pull dantheman213/rpcvg
```

#### Build image (if pulling from this repo)

``` 
docker build -t rpcvg .
```

NOTE: Use `rpcvg:latest` in the `run` command below instead if building from this repo

#### Run the container

##### Run container with single ingestion

```
docker run --rm -d --name rpcvg \
    -v /opt/rpcvg/cache:/cache \
    -v /opt/rpcvg/sources:/data \
    -v /opt/rpcvg/output:/output \
    dantheman213/rpcvg:latest \
    https://www.reddit.com/r/BetterEveryLoop/top/?t=year
```

##### Run container with multiple ingestions

```
docker ... dantheman213/rpcvg:latest \
    https://www.reddit.com/r/BetterEveryLoop/top/?t=year \
    https://www.reddit.com/r/gifs/top/?t=year \
    https://www.reddit.com/r/funny/top/?t=year
```

##### Running on Windows

Here are some example paths you can use:

```
-v C:\Users\YOURUSER\Desktop\rpcvg\cache:/cache
-v C:\Users\YOURUSER\Desktop\rpcvg\source:/data
-v C:\Users\YOURUSER\Desktop\rpcvg\output:/output
```
