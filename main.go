package main

import (
    "context"
    "fmt"
    "github.com/chromedp/cdproto/cdp"
    "log"
    "strings"

    "github.com/chromedp/chromedp"
)

func main() {
    fmt.Println("Starting ingestion process...")

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

    sourceUrl := "https://gfycat.com"
    for _, n := range nodes {
        url := n.AttributeValue("href")
        if strings.Index(url, sourceUrl) > -1 {
            log.Println(url)
        }
    }

    fmt.Println("COMPLETE!")
}
