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
    inputList := ""
    for _, file := range files {
        inputList += fmt.Sprintf("-i %s ", file)
    }

    runCommand(workDir, "ffmpeg", strings.Split(fmt.Sprintf("-f concat -safe 0 %s -movflags faststart -c:v copy %s", inputList, outputDir + "/export.mp4"), " "))
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

