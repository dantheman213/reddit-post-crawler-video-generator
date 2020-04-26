package main

import (
    "bytes"
    "context"
    "fmt"
    "github.com/chromedp/cdproto/dom"
    "github.com/chromedp/chromedp"
    "github.com/mvdan/xurls"
    "io"
    "log"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"
)

// TODO: put in config file?
var sources []string = []string {
    "https://gfycat.com/",
    "https://i.imgur.com/",
    //"https://preview.redd.it/",
}

func main() {
    if len(os.Args) <= 1 {
        fmt.Println("./rpcvg <reddit url>")
        fmt.Println("https://www.reddit.com/r/BetterEveryLoop/top/?t=week")
        os.Exit(0)
    }

    ingestionUrl := os.Args[1]

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

    var html *string = nil
    ctx, cancel := chromedp.NewContext(context.Background())
    defer cancel()

    // TODO: scroll down the page some?
    // https://github.com/chromedp/chromedp/issues/525
    err := chromedp.Run(
        ctx, chromedp.Navigate(ingestionUrl),
        //chromedp.WaitVisible("body"),
        chromedp.Sleep(5000 * time.Millisecond),
        chromedp.ActionFunc(func(ctx context.Context) error {
            node, err := dom.GetDocument().Do(ctx)
            if err != nil {
                return err
            }
            data, err := dom.GetOuterHTML().WithNodeID(node.NodeID).Do(ctx)
            html = &data
            return err
        }))

    if err != nil {
        fmt.Errorf("could not navigate to page: %v", err)
    }

    fmt.Println("Loaded web page.. looking for URLs...")

    rxStrict := xurls.Strict()
    rawUrls := rxStrict.FindAllString(*html, -1)
    filteredUrls := make([]string, 0)

    for _, s := range rawUrls {
        fmt.Println(s)

        for _, source := range sources {
            index := strings.Index(s, source)
            if index > -1 {
                url := s[index:len(s)]
                filteredUrls = append(filteredUrls, url)
            }
        }
    }

    count := len(filteredUrls)
    if count < 1 {
        fmt.Println("Didn't find any media to ingest!")
        os.Exit(0)
    }
    fmt.Printf("Found %d media items...\n", count)

    for _, url := range dedupeList(filteredUrls) {
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
        revisedFile := fmt.Sprintf("%s.REVISED.mp4", file[0:len(file) - 4])
        runCommand(workDir, "ffmpeg", strings.Split(fmt.Sprintf("-i %s -vf scale=w=1920:h=1080:force_original_aspect_ratio=decrease,pad=1920:1080:(ow-iw)/2:(oh-ih)/2:black,fps=60 -crf 18 -pix_fmt yuv420p -movflags faststart -c:v libx264 -c:a aac -f mp4 -y %s", file, revisedFile), " "))

        if _, err := f.WriteString(fmt.Sprintf("file '%s'\n", revisedFile)); err != nil {
            log.Println(err)
        }
    }

    runCommand(workDir, "ffmpeg", strings.Split(fmt.Sprintf("-f concat -safe 0 -i /tmp/list.txt -c copy -movflags faststart -f mp4 -y %s", outputDir + "/export.mp4"), " "))
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
