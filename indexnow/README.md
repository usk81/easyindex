# easyindex for Google indexing API <!-- omit in toc -->

- [Required](#required)
- [Install](#install)
- [Example](#example)
  - [Publish API](#publish-api)

## Required

- Google Indexing API
  - generate key
    - ref. https://www.bing.com/indexnow
  - hosting a text key file within your host
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
	"context"
	"fmt"

	"github.com/usk81/easyindex/indexnow"
	"github.com/usk81/easyindex/logger"
)

func main() {
	l, err := logger.New("debug")
	if err != nil {
		panic(err)
	}

	us := []string{
		"https://example.com/foo",
		"https://example.com/bar",
	}

	submit, err := indexnow.Submit(indexnow.Config{
		SearchEngineBaseURL: indexnow.Bing,
		Logger:              l,
		Logging:             true,
	})
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	us, skips := submit.Precheck(ctx, us)

	// ref. https://www.indexnow.org/documentation
	err = submit.Execute(indexnow.SubmitInput{
		Context:     ctx,
		Host:        "https://example.com",
		Key:         "5d87a32cd39c4162bbd580ffaa6b511f",
		KeyLocation: "https://example.com/5d87a32cd39c4162bbd580ffaa6b511f.txt",
		URLList:     us,
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("requests: %#v\n", us)
	fmt.Printf("skipRequests: %#v\n", skips)
}
```
