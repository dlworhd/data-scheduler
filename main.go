package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron"
	"golang.org/x/net/html"
)

type Item struct {
	Guid        string
	Title       string
	Thumbnail   string
	Description string
	Link        string
	Published   string
}

var ctx = context.Background()
var wg sync.WaitGroup

func main() {
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")

	if redisHost == "" {
		redisHost = "localhost"
	}

	if redisPort == "" {
		redisPort = "6379"
	}
	rdb := GetRedisClient(redisHost+":"+redisPort, "", 0)
	allChannels := GetAllChannels()

	c := cron.New()
	c.AddFunc("* 0, 12 * * *", func() {
		for key := range allChannels {
			clearItems(rdb, key)
		}

		// for key := range allChannels {
		// 	clearGuidAndItems(rdb, key)
		// }

		start := time.Now()

		InsertAllChannelItems(&wg, ctx, rdb, allChannels)
		wg.Wait()

		end := time.Now()
		//REDIS 두 개의 데이터를 쓴다. [key; channelName, value: recent guid], [key: channelName, value: []Item ]
		fmt.Println(end.Sub(start))
	})
	c.Start()
	for {

	}

}

func clearItems(rdb *redis.Client, key string) {
	rdb.Del(ctx, key+"_ITEM")
}

func clearGuidAndItems(rdb *redis.Client, key string) {
	rdb.Del(ctx, key+"_GUID")
	rdb.Del(ctx, key+"_ITEM")
}

func GetRedisClient(address string, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password, // no password set
		DB:       db,       // use default DB
	})
}

func GetAllChannels() map[string]string {
	/* Key  : Channel Name */
	/* Value: Channel Object  */
	channels := map[string]string{
		"TOSS":                    "https://toss.tech/rss.xml",
		"KAKAO":                   "https://tech.kakao.com/feed/",
		"GEEK_NEWS":               "https://feeds.feedburner.com/geeknews-feed",
		"KURLY":                   "http://thefarmersfront.github.io/feed.xml",
		"WOOWA_BROS":              "https://techblog.woowahan.com/feed/",
		"DANGGN":                  "https://medium.com/feed/daangn",
		"WATCHA":                  "https://medium.com/feed/watcha",
		"SOCAR":                   "https://tech.socarcorp.kr/feed.xml",
		"BANK_SALAD":              "https://blog.banksalad.com/rss.xml",
		"GOOGLE_DEVELOPERS_KOREA": "https://feeds.feedburner.com/GoogleDevelopersKorea",
		"TOAST_UI":                "https://ui.toast.com/rss.xml",
		"KAKAOPAY":                "https://tech.kakaopay.com/rss",
		"LOTTE_ON":                "https://techblog.lotteon.com/feed",
		"AWS":                     "https://aws.amazon.com/ko/blogs/tech/feed/",
		"SK_PLANET":               "https://techtopic.skplanet.com/rss.xml",
		"GMARKET":                 "https://ebay-korea.tistory.com/rss",
		"29CM":                    "https://medium.com/feed/@dev29cm",
		"HYPERCONNECT":            "https://hyperconnect.github.io/feed.xml",
		"GCCOMPANY":               "https://techblog.gccompany.co.kr/feed",
		"SCATTERLAB":              "https://tech.scatterlab.co.kr/feed.xml",
		"BUZZVIL":                 "https://tech.buzzvil.com/index.xml",
		"LINE":                    "https://techblog.lycorp.co.jp/ko/feed/index.xml",
		"SARAMIN":                 "https://saramin.github.io/feed.xml",
		"CLOUDVILLAINS":           "https://medium.com/feed/cloudvillains",
		"NETMARBLE":               "https://netmarble.engineering/feed/",
		"RIDI":                    "https://ridicorp.com/story-category/tech-blog/feed/",
		"PRND":                    "https://medium.com/feed/prnd",
		"TABLING":                 "https://techblog.tabling.co.kr/feed",
		"POSTYPE":                 "https://team.postype.com/rss",
		"11ST":                    "https://11st-tech.github.io/rss/",
		"NAVER_PLACE":             "https://medium.com/feed/naver-place-dev",
		"SQUARELAB":               "https://squarelab.co/feed.xml",
		"NHN_CLOUD":               "https://meetup.nhncloud.com/rss",
		"ZIGBANG":                 "https://medium.com/feed/zigbang",
		"COM2US":                  "https://on.com2us.com/wp-content/plugins/kboard/rss.php",
		"WANTED":                  "https://medium.com/feed/wantedjobs",
		"SPOQA":                   "https://spoqa.github.io/atom.xml",
		"CJ_ONSTYLE":              "https://medium.com/feed/cj-onstyle",
		"DEVSISTERS":              "https://tech.devsisters.com/rss.xml",
		"STARTLINK":               "https://startlink.blog/feed/",
		"HWAHAE":                  "https://blog.hwahae.co.kr/rss.xml",
		"YOGIYO":                  "https://techblog.yogiyo.co.kr/feed",
		"ESTSOFT":                 "https://blog.est.ai/feed.xml",
		"DRAMANCOMPANY":           "https://blog.dramancompany.com/feed/",
		"MUSINSA":                 "https://medium.com/feed/musinsa-tech",
		"TVING":                   "https://medium.com/feed/tving-team",
		"GITHUB":                  "https://github.blog/feed/",
		"GOOGLE":                  "https://developers.googleblog.com/feeds/posts/default",
		"NETFLIX":                 "https://netflixtechblog.com/feed",
		"PAYPAL":                  "https://slack.engineering/feed",
		"PINTEREST":               "https://medium.com/feed/@Pinterest_Engineering",
		"SPOTIFY":                 "https://engineering.atspotify.com/feed",
		"SLACK":                   "https://medium.com/feed/paypal-tech",
		"ZOOM":                    "https://medium.com/feed/zoom-developer-blog",
	}

	return channels
}

func GetRecentGUIDInRedis(ctx context.Context, rdb *redis.Client, channelName string) string {
	GUID, err := rdb.Get(ctx, channelName).Result()

	if err != nil {
		fmt.Println(err)
	}

	return GUID
}

func SetRecentGUID(ctx context.Context, rdb *redis.Client, channelName string, GUID string, expiration time.Duration) {
	rdb.Set(ctx, channelName, GUID, expiration).Err()
}

func SetRecentITEM(ctx context.Context, rdb *redis.Client, channelName string, serializedItem []byte, expiration time.Duration) {
	rdb.Set(ctx, channelName, serializedItem, expiration).Err()
}

func GetRecentGUIDInFeed(feed *gofeed.Feed) string {
	return feed.Items[0].GUID
}

func GetChannelItems(parser *gofeed.Parser, result map[string][]Item) {

}

func InsertAllChannelItems(wg *sync.WaitGroup, ctx context.Context, rdb *redis.Client, channels map[string]string) map[string][]Item {

	parser := gofeed.NewParser()
	result := map[string][]Item{}

	for channelName := range channels {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			items := []Item{}
			feed, err := parser.ParseURL(channels[name])
			if err != nil {
				return
			}

			html, _ := FetchHtml(feed.Link)
			thumbnail, err := ExtractThumbnail(html)
			if err != nil {

			}

			recentRGUID := GetRecentGUIDInRedis(ctx, rdb, name+"_GUID")
			recentFGUID := GetRecentGUIDInFeed(feed)

			if feed.Items[0].GUID == recentRGUID {
				fmt.Println("[No Updated Item] - " + name)
				return
			}

			for _, item := range feed.Items {
				newItem := Item{
					Guid:        item.GUID,
					Title:       item.Title,
					Description: item.Description,
					Thumbnail:   thumbnail,
					Link:        item.Link,
					Published:   item.Published,
				}
				fmt.Println(item.GUID)
				items = append(items, newItem)
			}
			jsonData, _ := json.Marshal(items)

			SetRecentGUID(ctx, rdb, name+"_GUID", recentFGUID, 0)
			SetRecentITEM(ctx, rdb, name+"_ITEM", jsonData, 0)
			result[name] = items
		}(channelName)
	}
	return result
}

// for _, feedArr := range rssArr {
// 	for _, feed := range feedArr {
// 		fmt.Println("GUID: ", feed.Guid)
// 		fmt.Println("Title: ", feed.Title)
// 		fmt.Println("Thumbnail: ", feed.Thumbnail)
// 		fmt.Println("Description : ", feed.Description)
// 		fmt.Println("Link: ", feed.Link)
// 		fmt.Println("Published: ", feed.Published)
// 		fmt.Printf("\n")
// 	}
// }

//NewItems를 해서 Item[title, description, link] 등을 가져온다.
//DB에 넣어야 되는데

// 채널(1) < 피드(N)
// 채널 이름은 Unique하게 KAKAOTECH, TOSSTECH 등 별칭은 카카오 테크, 토스 테크 등으로 설정하기

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
