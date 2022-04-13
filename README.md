# easyindex <!-- omit in toc -->

- [preinstall](#preinstall)
- [required](#required)
  - [install](#install)
  - [example](#example)
  - [use as CLI](#use-as-cli)
- [milestones](#milestones)

## preinstall

- create Google service account
- create credential json file for Google Indexing API
- add your service account as a site owner on search console

ref. [Google Search Central](https://developers.google.com/search/apis/indexing-api/v3/prereqs)

## required

- Google service account
- credential json file for Google Indexing API
- Go +1.17

### install

```
go get github.com/usk81/easyindex
```
### example

```go
import (
    "fmt"

    "github.com/usk81/easyindex"
    "github.com/usk81/easyindex/coordinator"
    "github.com/usk81/easyindex/logger"
)

func main() {
    cf := "./credential.json"
    l, err := logger.New("debug")
    if err != nil {
        panic(err)
    }
    s, err := coordinator.New(coordinator.Config{
        CredentialsFile: &cf,
        Logger:          l,
        Skip:            true,
    })
    if err != nil {
        panic(err)
    }
    rs := []coordinator.PublishRequest{
        {
            URL:              "http://example.com/foo",
            NotificationType: easyindex.NotificationTypeUpdated,
        },
        {
            URL:              "http://example.com/bar",
            NotificationType: easyindex.NotificationTypeDeleted,
        },
    }
    total, count, resp, skips, err := s.Publish(rs, limit)
    if err != nil {
        panic(err)
    }
    fmt.Printf("total: %d\n", total)
    fmt.Printf("count: %d\n", count)
    fmt.Printf("response: %#v\n", resp)
    fmt.Printf("skipRequests: %#v\n", skips)
}
```

### use as CLI

ref. https://github.com/usk81/easyindex-cli

## milestones

- API
  - [ ] getMetadata
  - publish
    - [x] basic
    - csv import 
- plugin
  - logger
    - [x] basic (uber/zap)
    - [ ] customize (other logger)
- CI/CD
  - [ ] unit tests
  - [ ] linter
  - [ ] auto review
  - [ ] release drafter
- others
  - [ ] flexible error handling
  - [ ] unit tests