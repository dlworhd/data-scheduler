package scheduler

import (
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html"
)

type Feed struct {
	Guid        string
	Title       string
	Thumbnail   string
	Description string
	Link        string
	Published   string
}

func ProceedFeed(fp *gofeed.Parser, rss string, wg *sync.WaitGroup, outerFeedArr [][]Feed, outIndex int) {
	innerWg := new(sync.WaitGroup)

	defer wg.Done()

	feeds, err := fp.ParseURL(rss)
	if err != nil {
		log.Printf("Error parsing URL %s: %v", rss, err)
		panic(err)
	}

	innerWg.Add(len(feeds.Items))

	var innerFeedArr []Feed

	for _, feed := range feeds.Items {
		go AddFeedItem(feed, innerWg, &innerFeedArr)
	}

	innerWg.Wait()
	(outerFeedArr)[outIndex] = innerFeedArr

}

func FetchHtml(url string) (string, error) {
	response, err := http.Get(url)

	if err != nil {
		return "", err
	}

	defer response.Body.Close()
	content, err := io.ReadAll(response.Body)

	if err != nil {
		return "", err
	}

	return string(content), nil

}

func ExtractThumbnail(htmlContent string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", err
	}

	var ThumbnailURL string
	var traverseFunc func(*html.Node)
	traverseFunc = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			for _, attr := range n.Attr {
				if attr.Key == "property" && attr.Val == "og:image" {
					for _, subAttr := range n.Attr {
						if subAttr.Key == "content" {
							ThumbnailURL = subAttr.Val
							return
						}
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverseFunc(c)
		}
	}
	traverseFunc(doc)

	return ThumbnailURL, nil
}

func AddFeedItem(feed *gofeed.Item, innerWg *sync.WaitGroup, feedArr *[]Feed) {
	defer innerWg.Done()
	html, err := FetchHtml(feed.Link)
	if err != nil {
		log.Printf("Error fetching HTML for %s: %v", feed.Link, err)
		panic(err)
	}
	if err != nil {
		log.Printf("Error extracting Thumbnail for %s: %v", feed.Link, err)
		panic(err)
	}
	Thumbnail, _ := ExtractThumbnail(html)

	f := Feed{
		Guid:        feed.GUID,
		Title:       feed.Title,
		Thumbnail:   Thumbnail,
		Description: feed.Description,
		Link:        feed.Link,
		Published:   feed.Published,
	}

	*feedArr = append(*feedArr, f)
}

func RedisProceedFeed(lastGuid string, fp *gofeed.Parser, rss string, wg *sync.WaitGroup, outerFeedArr [][]Feed, outIndex int) string {
	innerWg := new(sync.WaitGroup)

	defer wg.Done()

	feeds, err := fp.ParseURL(rss)
	if err != nil {
		log.Printf("Error parsing URL %s: %v", rss, err)
		panic(err)
	}

	innerWg.Add(len(feeds.Items))

	var innerFeedArr []Feed
	var recentGuid string
	for index, feed := range feeds.Items {
		if index == 0 {
			recentGuid = feed.GUID
			log.Println(recentGuid)
		}

		if feed.GUID != lastGuid {
			AddFeedItem(feed, innerWg, &innerFeedArr)
		}
	}

	innerWg.Wait()
	(outerFeedArr)[outIndex] = innerFeedArr

	return recentGuid
}
