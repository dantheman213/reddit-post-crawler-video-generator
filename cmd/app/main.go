package main

import (
    "bytes"
    "context"
    "fmt"
    "github.com/chromedp/cdproto/cdp"
    "github.com/chromedp/cdproto/dom"
    "github.com/chromedp/cdproto/network"
    "github.com/chromedp/cdproto/runtime"
    "github.com/chromedp/chromedp"
    "github.com/mvdan/xurls"
    "io"
    "io/ioutil"
    "log"
    "math"
    "math/rand"
    "os"
    "os/exec"
    "path/filepath"
    "regexp"
    "strings"
    "time"
)

// TODO: put in config file?
var sources []string = []string {
    "https://gfycat.com/",
    "https://i.imgur.com/",
    "https://www.redgifs.com/",
    "https://redgifs.com/",
    "https://preview.redd.it/",
    "https://v.redd.it/",
    "https://www.instagram.com/",
    "https://instagram.com/",
    "https://twitter.com/",
}

var workDir string = "/data"
var originalSourceDir string = workDir + "/original"
var revisedSourceDir string = workDir + "/revised"
var outputDir string = "/output"

var skipDownload bool = false
var skipNormalize bool = false

func main() {
    if len(os.Args) <= 1 {
        help()
    }

    readOptions()
    if !skipDownload && !skipNormalize {
        ingestURLsFromWebsites()
    }
    if !skipNormalize {
        normalizeVideos()
    }
    exportProduct()

    fmt.Println("COMPLETE!")
}

func ingestURLsFromWebsites() {
    ingestionUrls := make([]string, 0)
    for i, ingest := range os.Args {
        if i == 0 {
            continue
        }

        if strings.HasPrefix(ingest, "--") {
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

        // create list of actions starting with easily repeated actions (scrolling to bottom)
        actions := []chromedp.Action {
            chromedp.ActionFunc(func(ctx context.Context) error {
                _, exp, err := runtime.Evaluate(`window.scrollTo(0,document.body.scrollHeight);`).Do(ctx)
                if err != nil {
                    return err
                }
                if exp != nil {
                    return exp
                }
                return nil
            }),
            chromedp.Sleep(2000 * time.Millisecond),
        }
        for i := 0; i < 10; i++ {
            actions = append(actions, actions[0], actions[1])
        }

        // now add list of instructions to top of action list
        actions = append(actions, []chromedp.Action {
            SetCookie("over18", "yes", "www.reddit.com", "/", false, false),
            chromedp.Navigate(ingestionUrl),
            chromedp.Sleep(5000 * time.Millisecond),
        }...)

        // now at the bottom of action list
        actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
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

        if err := chromedp.Run(ctx, actions...); err != nil {
            fmt.Errorf("could not navigate to page: %v", err)
        }

        fmt.Printf("Loaded web page %s.. looking for URLs...\n", ingestionUrl)
        rawUrls := xurls.Strict.FindAllString(*html, -1)

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
}

func normalizeVideos() {
    // Get the list of files that was downloaded to the original sources
    files, _ := walkMatch(originalSourceDir, "*.mp4")
    fmt.Printf("Found  %d items to transcode...", len(files))
    // Normalize all the ingested videos into a common format: 1080p, 60fps, 16:9 aspect ratio without stretching image, pixel format, audio codec/bitrate/sample rate, etc.
    // Output videos into revised folder

    reg, err := regexp.Compile("[^a-zA-Z0-9]+")
    if err != nil {
        log.Fatal(err)
    }

    for i, file := range files {
        printPercentageDone(int64(i), int64(len(files)))

        interimFile := fmt.Sprintf("%s/%s.mp4", originalSourceDir, reg.ReplaceAllString(file[strings.LastIndex(file, "/") + 1:len(file) - 4], ""))
        if interimFile != file {
            fmt.Printf("Renaming file %s to remove bad characters for transcode normalization....\n", file)
            if err := os.Rename(file, interimFile); err != nil {
                log.Fatal(err)
            }
        }

        revisedFile := fmt.Sprintf("%s/%s.REVISED.mp4", revisedSourceDir, interimFile[strings.LastIndex(interimFile, "/") + 1:len(interimFile) - 4])
        runCommand(originalSourceDir, "ffmpeg", strings.Split(fmt.Sprintf(`-i %s -f lavfi -i anullsrc=cl=1 -vf scale=w=1920:h=1080:force_original_aspect_ratio=decrease,pad=1920:1080:(ow-iw)/2:(oh-ih)/2:black,fps=60 -c:v libx264 -crf 18 -pix_fmt yuv420p -shortest -c:a aac -ab 128k -ac 2 -ar 44100 -movflags faststart -f mp4 %s`, interimFile, revisedFile), " "))
    }
}

func exportProduct() {
    // Generate the exported final product which will concatenate all the revised videos together into one container
    raw := ""
    files, _ := walkMatch(revisedSourceDir, "*.mp4")
    fmt.Printf("Exporting %d items to final product...", len(files))
    for _, file := range files {
        raw += fmt.Sprintf("file '%s'\n", file)
    }
    ioutil.WriteFile("/tmp/list.txt", []byte(raw), os.ModePerm)

    exportFilePath := fmt.Sprintf("%s/export_%s.mp4", outputDir, time.Now().Format("20060102150405"))
    runCommand(revisedSourceDir, "ffmpeg", strings.Split(fmt.Sprintf("-f concat -safe 0 -i /tmp/list.txt -c:v copy -c:a copy -strict -2 -fflags +genpts -movflags faststart -f mp4 -y %s", exportFilePath), " "))
}

func readOptions() {
    if os.Args[1] == "--skip-download" {
        skipDownload = true
    } else if os.Args[1] == "--skip-normalize" {
        skipNormalize = true
    }
}

func help() {
    fmt.Println("Usage: ./rpcvg [options] <reddit subreddit/filter(s)>")
    fmt.Println("Example 1: ./rpcvg BetterEveryLoop,week")
    fmt.Println("Example 1: ./rpcvg BetterEveryLoop,week")
    fmt.Println("Example 2: ./rpcvg BetterEveryLoop,week,all")
    fmt.Printf("Example 3: ./rpcvg BetterEveryLoop,week funny,month gifs,week\n\n")
    fmt.Println("FILTERS:")
    fmt.Println("Duration: hour,day,week,month,year,all")
    fmt.Println("OPTIONS:")
    fmt.Println("--skip-download : Skip downloading assets and use whatever is in sources/original and begin normalization process")
    fmt.Println("--skip-normalize : Skip downloading and normalizing assets and use whatever is in sources/revised to export a final product")

    os.Exit(0)
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
