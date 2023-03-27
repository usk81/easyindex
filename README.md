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

- Google Indexing API
  - Google service account
  - credential json file for Google Indexing API
- IndexNow API
  - generate key
    - ref. https://www.bing.com/indexnow
  - hosting a text key file within your host
- Go +1.18

### install

```
go get github.com/usk81/easyindex
```
### example

- [Google Indexing API](google/README.md)
- [IndexNow API](indexnow/README.md)

### use as CLI

ref. https://github.com/usk81/easyindex-cli

## milestones

- API
  - [x] Google Indexing API
  - [x] IndexNow API
- plugin
  - logger
    - [x] basic (uber/zap)
    - [ ] customize (other logger)
- CI/CD
  - [x] unit tests
  - [x] linter
  - [x] auto review
  - [x] release drafter
