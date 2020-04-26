FROM ubuntu:20.04

WORKDIR /tmp
RUN apt-get update

# Install Google Chrome
RUN apt-get install -y curl libappindicator1 fonts-liberation
RUN curl -o chrome.deb https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb
RUN apt install -y /tmp/chrome.deb


# Install ffmpeg
RUN apt-get install -y ffmpeg

# Install Golang
RUN curl -o go.tar.gz https://dl.google.com/go/go1.14.2.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go.tar.gz
ENV PATH ${PATH}:/usr/local/go/bin

# Install youtube-dl
RUN apt-get install -y python3 && \
    ln -s /usr/bin/python3 /usr/bin/python
RUN curl -L https://yt-dl.org/downloads/latest/youtube-dl -o /usr/local/bin/youtube-dl && \
    chmod a+rx /usr/local/bin/youtube-dl
RUN youtube-dl -U

# App
RUN apt-get install -y git
RUN go get -t github.com/chromedp/chromedp
COPY main.go /usr/local/.

ENTRYPOINT ["go", "run", "/usr/local/main.go"]