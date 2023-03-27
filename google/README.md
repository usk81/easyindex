# easyindex for Google indexing API <!-- omit in toc -->

- [Required](#required)
- [Install](#install)
- [Example](#example)
  - [Publish API](#publish-api)
- [milestones](#milestones)

## Required

- Google Indexing API
  - Google service account
  - credential json file for Google Indexing API
- Go +1.18

## Install

```
go get github.com/usk81/easyindex
```

## Example

### Publish API

```go
package main

import (
    "fmt"

	"github.com/usk81/easyindex"
    "github.com/usk81/easyindex/google"
    "github.com/usk81/easyindex/logger"
	"google.golang.org/api/indexing/v3"
)

func main() {
    cf := "./credential.json"
    l, err := logger.New("debug")
    if err != nil {
        panic(err)
    }
    publish, err := google.Publish(goole.Config{
		Quota: google.PublishAPIDefaultQuota,
        CredentialsFile: &cf,
        Logger:          l,
        Logging:         true,
    })
    if err != nil {
        panic(err)
    }
    rs := []google.PublishRequest{
        {
            URL:              "http://example.com/foo",
            NotificationType: google.NotificationTypeUpdated,
        },
        {
            URL:              "http://example.com/bar",
            NotificationType: google.NotificationTypeDeleted,
        },
    }

    // precheck: Skip if the web page returns a status code other than 2xx.
	rs, skips, err := publish.Precheck(rs)
	if err != nil {
		panic(err)
	}

	resps := []*indexing.PublishUrlNotificationResponse{}
	for _, r := range rs {
		resp, skip, err := publish.Execute(r)
		if skip {
			skips = append(skips, google.SkipedPublishRequest{
				URL: r.URL,
				NotificationType: r.NotificationType,
				Reason: ErrExceededQuota,
			})
		} else if err != nil {
			panic(err)
		} else {
			resps = append(resps, resp)
		}
	}
    fmt.Printf("response: %#v\n", resps)
    fmt.Printf("skipRequests: %#v\n", skips)
}
```

## milestones

- [x] publish
- [ ] getMetadata
