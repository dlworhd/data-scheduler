package main

import (
	"context"
	"fmt"
	"sync"

	scheduler "github.com/dlworhd/data-scheduler"
	"github.com/go-redis/redis"

	"github.com/mmcdole/gofeed"
)

var ctx = context.Background()

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	wg := new(sync.WaitGroup)
	fp := gofeed.NewParser()

	rssUrls := []string{
		"https://toss.tech/rss.xml",
	}

	wg.Add(len(rssUrls))

	var outerFeedArr [][]scheduler.Feed

	for range rssUrls {
		outerFeedArr = append(outerFeedArr, nil)
	}

	for i, rss := range rssUrls {
		lastGuidInDb, err := rdb.Get(ctx, rss).Result()
		if err != nil {
			panic(err)
		}
		recentGuid := RedisProceedFeed(lastGuidInDb, fp, rss, wg, outerFeedArr, i)

		err = rdb.Set(ctx, rss, recentGuid, 0).Err()
		if err != nil {
			panic(err)
		}

		lastGuidInDb, err = rdb.Get(ctx, rss).Result()
		fmt.Print(lastGuidInDb)
	}

	for _, innerFeedArr := range outerFeedArr {
		for _, feed := range innerFeedArr {
			fmt.Println("GUID: ", feed.Guid)
			fmt.Println("Title: ", feed.Title)
			fmt.Println("Thumbnail: ", feed.Thumbnail)
			fmt.Println("Description : ", feed.Description)
			fmt.Println("Link: ", feed.Link)
			fmt.Println("Published: ", feed.Published)
			fmt.Printf("\n")
		}
	}

	wg.Wait()

}
