package indexnow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/usk81/easyindex"
	"github.com/usk81/toolkit/slice"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	Bing   = "https://www.bing.com"
	Seznam = "https://seznam.cz"
	Yandex = "https://yandex.com"
)

type (
	SubmitInput struct {
		Context     context.Context
		Host        string
		Key         string
		KeyLocation string
		URLList     []string
	}

	submitAPIRequest struct {
		Host        string   `json:"host"`
		Key         string   `json:"key"`
		KeyLocation string   `json:"keyLocation"`
		URLList     []string `json:"urlList"`
	}

	// Submitter ...
	Submitter interface {
		Precheck(ctx context.Context, input []string) (requests []string, skips []SkipedPublishRequest)
		Execute(input SubmitInput) (err error)
	}

	Config struct {
		// SearchEngineBaseURL
		//   e.g. https://www.bing.com
		SearchEngineBaseURL string

		// client
		Client *http.Client

		// Logger is logger instance
		Logger *zap.Logger

		// Logging is the flag if outputs logs
		Logging bool

		// Quota is Publish API requests limit per day
		Quota *int
	}

	submitImpl struct {
		apiEndpoint string

		// client
		client *http.Client

		// logger is logger instance
		logger *zap.Logger

		// Logging is the flag if outputs logs
		logging bool

		// Quota is Publish API requests limit per day
		quota *int
	}

	SkipedPublishRequest struct {
		URL    string
		Reason error
	}
)

func MustSubmit(conf Config) Submitter {
	s, err := Submit(conf)
	if err != nil {
		panic((err))
	}
	return s
}

func Submit(conf Config) (Submitter, error) {
	client := conf.Client
	if client == nil {
		client = http.DefaultClient
	}

	u, err := url.Parse(conf.SearchEngineBaseURL)
	if err != nil {
		errorLog(
			conf.Logging, conf.Logger,
			"submit.Initialize",
			zap.String("action", "url.Parse"),
			zap.String("search engine", conf.SearchEngineBaseURL),
			zap.Error(err),
		)
		return nil, err
	}
	u.Path = "/indexnow"
	endpoint := u.String()

	quota := conf.Quota
	if quota == nil && u.Scheme+"://"+u.Host == Seznam {
		*quota = 500
	}

	return &submitImpl{
		apiEndpoint: endpoint,
		client:      client,
		logger:      conf.Logger,
		logging:     conf.Logging,
		quota:       quota,
	}, nil
}

func (i *submitImpl) Precheck(ctx context.Context, input []string) (requests []string, skips []SkipedPublishRequest) {
	if ctx == nil {
		ctx = context.Background()
	}
	if i.quota == nil {
		return i.precheckWithoutQuota(ctx, input)
	}
	return i.precheckWithQuota(ctx, input)
}

func (i *submitImpl) Execute(input SubmitInput) (err error) {
	if input.Context == nil {
		input.Context = context.Background()
	}
	if i.quota == nil {
		return i.executeWithoutQuota(input)
	}
	return i.executeWithQuota(input)
}

func (i *submitImpl) precheckWithQuota(ctx context.Context, input []string) (requests []string, skips []SkipedPublishRequest) {
	remain := *i.quota
	for _, u := range input {
		if remain <= 0 {
			skips = append(skips, SkipedPublishRequest{
				URL:    u,
				Reason: easyindex.ErrExceededQuota,
			})
		} else {
			resp, err := i.crawl(ctx, u)
			if err != nil {
				skips = append(skips, SkipedPublishRequest{
					URL:    u,
					Reason: err,
				})
			} else if resp.StatusCode > 300 {
				// error is returned　If status code is not 2xx. Includes redirects.
				skips = append(skips, SkipedPublishRequest{
					URL:    u,
					Reason: errors.New(resp.Status),
				})
			}
			remain--
		}
	}
	return
}

func (i *submitImpl) precheckWithoutQuota(ctx context.Context, input []string) (requests []string, skips []SkipedPublishRequest) {
	for _, u := range input {
		resp, err := i.crawl(ctx, u)
		if err != nil {
			skips = append(skips, SkipedPublishRequest{
				URL:    u,
				Reason: err,
			})
		} else if resp.StatusCode > 300 {
			// error is returned　If status code is not 2xx. Includes redirects.
			skips = append(skips, SkipedPublishRequest{
				URL:    u,
				Reason: errors.New(resp.Status),
			})
		}
	}
	return
}

func (i *submitImpl) executeWithQuota(input SubmitInput) (err error) {
	// You can submit up to 10,000 URLs per post, mixing http and https URLs if needed.
	//   ref. https://www.indexnow.org/documentation
	uss, err := slice.Chunk(input.URLList, 10000)
	if err != nil {
		i.error("submit.executeWithQuota", zap.String("action", "chunk url list"), zap.Error(err))
		return
	}
	quota := *i.quota
	for _, us := range uss {
		if quota <= 0 {
			return easyindex.ErrExceededQuota
		}
		r := submitAPIRequest{
			Host:        input.Host,
			Key:         input.Key,
			KeyLocation: input.KeyLocation,
			URLList:     us,
		}
		var buf bytes.Buffer
		if err = json.NewEncoder(&buf).Encode(r); err != nil {
			i.error("submit.executeWithQuota", zap.String("action", "create request body"), zap.Error(err))
			return
		}
		req, err := http.NewRequestWithContext(input.Context, http.MethodPost, i.apiEndpoint, &buf)
		if err != nil {
			i.error("submit.executeWithQuota", zap.String("action", "create request"), zap.Error(err))
			return err
		}
		resp, err := i.client.Do(req)
		quota--
		i.quota = &quota
		if err != nil {
			i.error("submit.executeWithQuota", zap.String("action", "send request"), zap.Error(err))
			return err
		}
		if resp.StatusCode >= 300 {
			if resp.StatusCode == http.StatusTooManyRequests {
				err = easyindex.ErrExceededQuota
			} else {
				err = errors.New(resp.Status)
			}
			i.error("submit.executeWithQuota", zap.String("action", "send request"), zap.Error(err))
			return err
		}
	}
	return
}

func (i *submitImpl) executeWithoutQuota(input SubmitInput) (err error) {
	// You can submit up to 10,000 URLs per post, mixing http and https URLs if needed.
	//   ref. https://www.indexnow.org/documentation
	uss, err := slice.Chunk(input.URLList, 10000)
	if err != nil {
		i.error("submit.executeWithoutOuota", zap.String("action", "chunk url list"), zap.Error(err))
		return
	}
	for _, us := range uss {
		r := submitAPIRequest{
			Host:        input.Host,
			Key:         input.Key,
			KeyLocation: input.KeyLocation,
			URLList:     us,
		}
		var buf bytes.Buffer
		if err = json.NewEncoder(&buf).Encode(r); err != nil {
			i.error("submit.executeWithoutOuota", zap.String("action", "create request body"), zap.Error(err))
			return
		}
		req, err := http.NewRequestWithContext(input.Context, http.MethodPost, i.apiEndpoint, &buf)
		if err != nil {
			i.error("submit.executeWithoutOuota", zap.String("action", "create request"), zap.Error(err))
			return err
		}
		resp, err := i.client.Do(req)
		if err != nil {
			i.error("submit.executeWithoutOuota", zap.String("action", "send request"), zap.Error(err))
			return err
		}
		if resp.StatusCode >= 300 {
			if resp.StatusCode == http.StatusTooManyRequests {
				err = easyindex.ErrExceededQuota
			} else {
				err = errors.New(resp.Status)
			}
			i.error("submit.executeWithoutOuota", zap.String("action", "send request"), zap.Error(err))
			return err
		}
	}
	return
}

func (i *submitImpl) error(msg string, fields ...zapcore.Field) {
	errorLog(i.logging, i.logger, msg, fields...)
}

func (i *submitImpl) crawl(ctx context.Context, u string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := i.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func errorLog(logging bool, logger *zap.Logger, msg string, fields ...zapcore.Field) {
	if logging && logger != nil {
		logger.Error(msg, fields...)
	}
}
