package main

import (
    "bytes"
    "context"
    "fmt"
    "github.com/chromedp/cdproto/cdp"
    "io"
    "log"
    "os"
    "os/exec"
    "path/filepath"
    "strings"

    "github.com/chromedp/chromedp"
)

func main() {
    fmt.Println("Starting ingestion process...")
    workDir := "/data"
    outputDir := "/output"

    if _, err := os.Stat(workDir); os.IsNotExist(err) {
        _ = os.Mkdir(workDir, os.ModePerm)
    } else {
        runCommand(workDir, "find", strings.Split(fmt.Sprintf("%s -type f -name \"*.mp4\" -delete", workDir), " "))
    }

    if _, err := os.Stat(outputDir); os.IsNotExist(err) {
        _ = os.Mkdir(outputDir, os.ModePerm)
    }

    // create context
    ctx, cancel := chromedp.NewContext(context.Background())
    defer cancel()

    var nodes []*cdp.Node

    err := chromedp.Run(ctx,
        chromedp.Navigate("https://www.reddit.com/r/BetterEveryLoop/top/?t=week"),
        chromedp.Nodes("a", &nodes))

    if err != nil {
        fmt.Errorf("could not navigate to page: %v", err)
    }

    // TODO
    sourceUrl := "https://gfycat.com"

    urls := make([]string, 0)
    for _, n := range nodes {
        s := n.AttributeValue("href")
        index := strings.Index(s, sourceUrl)
        if index > -1 {
            url := s[index:len(s)]
            urls = append(urls, url)
        }
    }

    fmt.Printf("Found %d media items...", len(urls))

    for _, url := range dedupeList(urls) {
        fmt.Printf("Attempting %s\n", url)
        runCommand(workDir, "youtube-dl", strings.Split(fmt.Sprintf("--no-check-certificate --prefer-ffmpeg --restrict-filenames %s", url), " "))
    }

    files, _ := walkMatch(workDir, "*.mp4")

    runCommand("/tmp", "rm", strings.Split("-fv /tmp/list.txt", " "))
    f, err := os.OpenFile("/tmp/list.txt",
        os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Println(err)
    }
    defer f.Close()

    for _, file := range files {
        if _, err := f.WriteString(fmt.Sprintf("file '%s'\n", file)); err != nil {
            log.Println(err)
        }
    }




    runCommand(workDir, "ffmpeg", strings.Split(fmt.Sprintf("-f concat -safe 0 -i /tmp/list.txt -aspect 16:9 -vf scale=w=1920:h=1080,pad=1920:1080:0:0:black,fps=60 -crf 18 -pix_fmt yuv420p -movflags faststart -c:v libx264 -c:a aac -f mp4 -y %s", outputDir + "/export.mp4"), " "))


//    runCommand(workDir, "ffmpeg", strings.Split(fmt.Sprintf("-f concat -safe 0 -i /tmp/list.txt -filter_complex scale=w=1920:h=1080:force_original_aspect_ratio=decrease,fps=60 -video_track_timescale 60 -crf 18 -pix_fmt yuv420p -movflags faststart -c:v libx264 -c:a copy -f mp4 -y %s", outputDir + "/export.mp4"), " "))


//    runCommand(workDir, "ffmpeg", strings.Split(fmt.Sprintf("-f concat -safe 0 -i /tmp/list.txt -filter_complex scale=w=1920:h=1080:force_original_aspect_ratio=decrease,fps=30 -vsync 1 -crf 18 -pix_fmt yuv420p -movflags faststart -c:v libx264 -c:a copy -f mp4 -y %s", outputDir + "/export.mp4"), " "))
    //runCommand(workDir, "ffmpeg", strings.Split(fmt.Sprintf("-f concat -safe 0 -i /tmp/list.txt -filter_complex scale=w=1920:h=1080,fps=30 -crf 18 -pix_fmt yuv420p -movflags faststart -c:v libx264 -c:a copy -f mp4 -y %s", outputDir + "/export.mp4"), " "))


    //runCommand(workDir, "ffmpeg", strings.Split(fmt.Sprintf("-f concat -safe 0 -i /tmp/list.txt -filter_complex scale=w=1920:h=1080:force_original_aspect_ratio=1,transpose=1,fps=30 -crf 18 -pix_fmt yuv420p -movflags faststart -c:v libx264 -c:a copy -s 1280*720 -f mp4 -y %s", outputDir + "/export.mp4"), " "))
    //runCommand(workDir, "ffmpeg", strings.Split(fmt.Sprintf("-f concat -safe 0 -i /tmp/list.txt -vf scale=w=1920:h=1080:force_original_aspect_ratio=1,pad=1920:1080:(ow-iw)/2:(oh-ih)/2 -crf 18 -pix_fmt yuv420p -movflags faststart -c:v libx264 -c:a copy -s 1280*720 -f mp4 -r 30 -y %s", outputDir + "/export.mp4"), " "))
    fmt.Println("COMPLETE!")
}

func runCommand(dir, command string, args []string) {
    cmd := exec.Command(command, args...)
    cmd.Dir = dir

    var stdBuffer bytes.Buffer
    mw := io.MultiWriter(os.Stdout, &stdBuffer)

    cmd.Stdout = mw
    cmd.Stderr = mw

    if err := cmd.Run(); err != nil {
        log.Panic(err)
    }

    log.Println(stdBuffer.String())
}

func dedupeList(s []string) []string {
    if len(s) <= 1 {
        return s
    }

    result := []string{}
    seen := make(map[string]struct{})

    for _, val := range s {
        if _, ok := seen[val]; !ok {
            result = append(result, val)
            seen[val] = struct{}{}
        }
    }

    return result
}

func walkMatch(root, pattern string) ([]string, error) {
    var matches []string
    err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if info.IsDir() {
            return nil
        }
        if matched, err := filepath.Match(pattern, filepath.Base(path)); err != nil {
            return err
        } else if matched {
            matches = append(matches, path)
        }
        return nil
    })
    if err != nil {
        return nil, err
    }
    return matches, nil
}

