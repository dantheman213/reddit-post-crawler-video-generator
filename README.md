# reddit-post-crawler-video-generator

Automatically generate compilation videos from popular Reddit posts.

### Getting Started

Grab the most popular content from [r/BetterEveryLoop](https://www.reddit.com/r/BetterEveryLoop/top/?t=month) in the last month and make a video out of it:

```
docker run --rm -d --name rpcvg \
    -v /opt/rpcvg/cache:/cache \
    -v /opt/rpcvg/sources:/data \
    -v /opt/rpcvg/output:/output \
    dantheman213/rpcvg:latest \
    BetterEveryLoop,month
```

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

### Additional Details

This software can be found on [DockerHub](https://hub.docker.com/r/dantheman213/rpcvg).

```
docker pull dantheman213/rpcvg:latest
```

### Examples

```
./rpcvg <reddit subreddit/highlight-duration>
ex: ./rpcvg BetterEveryLoop,week

Highlight Duration: hour,day,week,month,year,all
  - Example: Content from the last [week] only.
```

##### Run with multiple ingestions

All videos from all subs will be aggregated into a single comnpilation video.

```
docker ... dantheman213/rpcvg:latest \
    gifs,month \
    BetterEveryLoop,month \
    NatureGifs,all
```

##### Monitor Logs

Watch the ingestion, encoding, and merging process in real-time:

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

### Development

#### Build image (if pulling from this repo)

``` 
docker build -t test .
docker run test:latest
```
