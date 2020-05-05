# reddit-post-crawler-video-generator

Generate a single video from popular subreddits containing videos and GIFs and aggregate them into a video.

### How It Works

This Golang app works with Chrome (headless), ffmpeg, and youtube-dl wrapped in a Docker image to ingest, process, and produce content from your favorite Reddit subs.

Here's a break-down of what happens in the script:

1. Run Chrome headless and query target Reddit subs for URLs of interest
2. Pass URLs to `youtube-dl` to see if they can be downloaded; download all content before continuning
3. Videos will be in various native formats -- normalize them to:
  - 1080p, x264
  - Keep source aspect ratio by cropping evenly into 16:9 target format; add black bars if necessary
  - Add silent audio if there was no audio (e.g. GIFs)
  - AAC, 2 channels
4. Merge normalized videos into a single video

### Getting Started

#### Get this Docker image from DockerHub

Check out the image on DockerHub at [dantheman213/rpcvg](https://hub.docker.com/repository/docker/dantheman213/rpcvg) or pull it:

```
docker pull dantheman213/rpcvg
```

#### Build image (if pulling from this repo)

``` 
docker build -t rpcvg .
```

NOTE: Use `rpcvg:latest` in the `run` command below instead if building from this repo

#### Run the container

##### Usage

```
./rpcvg <reddit subreddit/highlight-duration>
ex: ./rpcvg BetterEveryLoop,week

Highlight Duration: hour,day,week,month,year,all
  - Example: Content from the last [week] only.
```

##### Run container with single ingestion

```
docker run --rm -d --name rpcvg \
    -v /opt/rpcvg/cache:/cache \
    -v /opt/rpcvg/sources:/data \
    -v /opt/rpcvg/output:/output \
    dantheman213/rpcvg:latest \
    gifs,month
```

##### Run container with multiple ingestions

This will aggregate all videos from all subs into one video.

```
docker ... dantheman213/rpcvg:latest \
    gifs,month \
    BetterEveryLoop,month \
    NatureGifs,all
```

##### Watch the ingestion, encoding, and mering process in real-time
```
docker logs -f rpcvg
```

##### Running on Windows

Here are some example paths you can use:

```
-v C:\Users\YOURUSER\Desktop\rpcvg\cache:/cache
-v C:\Users\YOURUSER\Desktop\rpcvg\source:/data
-v C:\Users\YOURUSER\Desktop\rpcvg\output:/output
```
