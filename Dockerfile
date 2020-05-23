FROM ubuntu:20.04 as staging

WORKDIR /tmp
RUN apt-get update

# Install ffmpeg
RUN apt-get install -y ffmpeg

# Install Google Chrome
RUN apt-get install -y curl libappindicator1 fonts-liberation
RUN curl -o chrome.deb https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb
RUN apt install -y /tmp/chrome.deb

# Install youtube-dl
RUN apt-get install -y python3 && \
    ln -s /usr/bin/python3 /usr/bin/python
RUN curl -L https://yt-dl.org/downloads/latest/youtube-dl -o /usr/local/bin/youtube-dl && \
    chmod a+rx /usr/local/bin/youtube-dl
RUN youtube-dl -U

FROM golang:1.14 as workspace

WORKDIR /go/src/app
COPY . .

RUN make deps
RUN make

# Bundle app
FROM staging as release
COPY --from=workspace /go/src/app/bin/rpcvg /usr/bin/rpcvg
RUN chmod +x /usr/bin/rpcvg

ENTRYPOINT ["/usr/bin/rpcvg"]
