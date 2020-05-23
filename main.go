package main

import (
    "bytes"
    "context"
    "fmt"
    "github.com/chromedp/cdproto/cdp"
    "github.com/chromedp/cdproto/dom"
    "github.com/chromedp/cdproto/network"
    "github.com/chromedp/chromedp"
    "github.com/mvdan/xurls"
    "io"
    "log"
    "math"
    "math/rand"
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
        fmt.Println("Usage: ./rpcvg <reddit subreddit/filter(s)>")
        fmt.Println("Example 1: ./rpcvg BetterEveryLoop,week")
        fmt.Println("Example 2: ./rpcvg BetterEveryLoop,week,all")
        fmt.Printf("Example 3: ./rpcvg BetterEveryLoop,week funny,month gifs,week\n\n")
        fmt.Println("OPTIONS:")
        fmt.Println("Duration: hour,day,week,month,year,all")
        os.Exit(0)
    }

    ingestionUrls := make([]string, 0)
    for i, ingest := range os.Args {
        if i == 0 {
            continue
        }

        parts := strings.Split(ingest, ",")
        subreddit := parts[0]

        for k, filter := range parts {
            if k == 0 {
                continue
            }

            url := fmt.Sprintf("https://www.reddit.com/r/%s/top/?t=%s", subreddit, strings.ToLower(filter))
            ingestionUrls = append(ingestionUrls, url)
        }
    }

    fmt.Println("Starting ingestion process...")
    filteredUrls := make([]string, 0)

    workDir := "/data"
    originalSourceDir := workDir + "/original"
    revisedSourceDir := workDir + "/revised"
    outputDir := "/output"

    // create dirs that need to exist
    if _, err := os.Stat(workDir); os.IsNotExist(err) {
        _ = os.Mkdir(workDir, os.ModePerm)
    }

    if _, err := os.Stat(originalSourceDir); os.IsNotExist(err) {
        _ = os.Mkdir(originalSourceDir, os.ModePerm)
    }

    if _, err := os.Stat(revisedSourceDir); os.IsNotExist(err) {
        _ = os.Mkdir(revisedSourceDir, os.ModePerm)
    }

    if _, err := os.Stat(outputDir); os.IsNotExist(err) {
        _ = os.Mkdir(outputDir, os.ModePerm)
    }

    fmt.Println("Clearing work cache....")
    runCommand(workDir, "find", strings.Split(fmt.Sprintf("%s -type f -name *.mp4 -delete", workDir), " "))
    runCommand("/tmp", "rm", strings.Split("-fv /tmp/list.txt", " "))

    for _, ingestionUrl := range ingestionUrls {
        var html *string = nil
        //var jsEval *string = nil
        ctx, cancel := chromedp.NewContext(context.Background())
        defer cancel()

        // TODO: scroll down the page some?
        // https://github.com/chromedp/chromedp/issues/525
        // scrolling currently doesn't work
        err := chromedp.Run(
            ctx,
            SetCookie("over18", "yes", "www.reddit.com", "/", false, false),
            chromedp.Navigate(ingestionUrl),
            chromedp.Sleep(5000 * time.Millisecond),
            //chromedp.Evaluate(`window.scrollTo(0,document.body.scrollHeight);`, &jsEval),
            chromedp.Sleep(1000 * time.Millisecond),
            //chromedp.Evaluate(`window.scrollTo(0,document.body.scrollHeight);`, &jsEval),
            chromedp.Sleep(2000 * time.Millisecond),
            chromedp.ActionFunc(func(ctx context.Context) error {
                node, err := dom.GetDocument().Do(ctx)
                if err != nil {
                    return err
                }

                data, err := dom.GetOuterHTML().WithNodeID(node.NodeID).Do(ctx)
                if err != nil {
                    return err
                }

                html = &data
                return err
            }))

        if err != nil {
            fmt.Errorf("could not navigate to page: %v", err)
        }

        fmt.Printf("Loaded web page %s.. looking for URLs...\n", ingestionUrl)

        rxStrict := xurls.Strict()
        rawUrls := rxStrict.FindAllString(*html, -1)

        for _, s := range rawUrls {
            for _, source := range sources {
                index := strings.Index(s, source)
                if index > -1 {
                    url := s[index:len(s)]
                    filteredUrls = append(filteredUrls, url)
                }
            }
        }
    }

    count := len(filteredUrls)
    if count < 1 {
        fmt.Println("Didn't find any media to ingest!")
        os.Exit(0)
    }
    fmt.Printf("Found %d media items...\n", count)

    fmt.Println("Checking for any updates to youtube-dl...")
    runCommand(originalSourceDir, "youtube-dl", []string {"-U"})

    // Go through each filteredUrl and download the video using youtube-dl
    deduped := dedupeList(filteredUrls)
    for i, url := range deduped {
        fmt.Printf("Attempting %s\n", url)
        time.Sleep(time.Duration(random(1500, 6500)) * time.Millisecond) // make sure external providers don't throttle your ingestion
        runCommand(originalSourceDir, "youtube-dl", strings.Split(fmt.Sprintf("--cache-dir /cache --no-check-certificate --prefer-ffmpeg --restrict-filenames %s", url), " "))
        printPercentageDone(int64(i), int64(len(deduped)))
    }

    // Get the list of files that was downloaded to the original sources
    files, _ := walkMatch(originalSourceDir, "*.mp4")
    f, err := os.OpenFile("/tmp/list.txt",
        os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Println(err)
    }
    defer f.Close()

    // Normalize all the ingested videos into a common format: 1080p, 60fps, 16:9 aspect ratio without stretching image, pixel format, audio codec/bitrate/sample rate, etc.
    // Output videos into revised folder
    for i, file := range files {
        printPercentageDone(int64(i), int64(len(files)))

        revisedFile := fmt.Sprintf("%s/%s.REVISED.mp4", revisedSourceDir, file[strings.LastIndex(file, "/") + 1:len(file) - 4])
        runCommand(originalSourceDir, "ffmpeg", strings.Split(fmt.Sprintf("-i %s -f lavfi -i anullsrc=cl=1 -vf scale=w=1920:h=1080:force_original_aspect_ratio=decrease,pad=1920:1080:(ow-iw)/2:(oh-ih)/2:black,fps=60 -c:v libx264 -preset:v slow -crf 18 -pix_fmt yuv420p -shortest -c:a aac -ab 128k -ac 2 -ar 44100 -movflags faststart -f mp4 -y %s", file, revisedFile), " "))

        // check if ffmpeg created the asset as we expected
        if _, err := os.Stat(revisedFile); !os.IsNotExist(err) {
            fmt.Printf("Revised file %s exists...\n", revisedFile)

            // if exists, make sure the file is at least 800KB, otherwise likely corrupted, garbage, or a copyright notice clip
            size, err := getFileSizeInBytes(revisedFile)
            if err == nil && size >= 800000 {
                // Write revised file into the file concat muxer list
                fmt.Printf("Revised file %s has size %d.. adding to list...\n", revisedFile, size)
                if _, err := f.WriteString(fmt.Sprintf("file '%s'\n", revisedFile)); err != nil {
                    log.Println(err)
                }
            } else {
                fmt.Printf("Detected and removing possible garbage at %s...", revisedFile)
                _ = os.Remove(revisedFile)
            }
        }
    }

    // Generate the exported final product which will concatenate all the revised videos together into one container
    exportFilePath := fmt.Sprintf("%s/export_%s.mp4", outputDir, time.Now().Format("20060102150405"))
    runCommand(revisedSourceDir, "ffmpeg", strings.Split(fmt.Sprintf("-f concat -safe 0 -i /tmp/list.txt -c:v copy -c:a copy -strict -2 -fflags +genpts -movflags faststart -f mp4 -y %s", exportFilePath), " "))
    fmt.Println("COMPLETE!")
}

func printPercentageDone(current, max int64) {
    fmt.Printf("\n\n\nOperation is %.1f%% complete!\n\n\n", math.Abs(float64(current) / float64(max)) * 100)
}

func runCommand(dir, command string, args []string) error {
    cmd := exec.Command(command, args...)
    cmd.Dir = dir

    var stdBuffer bytes.Buffer
    mw := io.MultiWriter(os.Stdout, &stdBuffer)

    cmd.Stdout = mw
    cmd.Stderr = mw

    if err := cmd.Run(); err != nil {
        return err
    }

    log.Println(stdBuffer.String())
    return nil
}

func SetCookie(name, value, domain, path string, httpOnly, secure bool) chromedp.Action {
    return chromedp.ActionFunc(func(ctx context.Context) error {
        expr := cdp.TimeSinceEpoch(time.Now().Add(180 * 24 * time.Hour))
        success, err := network.SetCookie(name, value).
            WithExpires(&expr).
            WithDomain(domain).
            WithPath(path).
            WithHTTPOnly(httpOnly).
            WithSecure(secure).
            Do(ctx)
        if err != nil {
            return err
        }
        if !success {
            return fmt.Errorf("could not set cookie %s", name)
        }
        return nil
    })
}


func dedupeList(s []string) []string {
    if len(s) <= 1 {
        return s
    }

    result := []string{}
    seen := make(map[string]struct{})

    for _, val := range s {
        normalizedStr := strings.TrimSpace(strings.ToLower(val))
        if _, ok := seen[normalizedStr]; !ok {
            result = append(result, val) // use original filename but just check if there are no dupes on case
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

func random(min, max int) int {
    return rand.Intn(max - min) + min
}

func getFileSizeInBytes(filepath string) (int64, error) {
    fi, err := os.Stat(filepath)
    if err != nil {
        return 0, err
    }
    return fi.Size(), nil
}
