package google

import (
	"context"
	"errors"
	"net/http"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/oauth2"
	"google.golang.org/api/indexing/v3"

	"github.com/usk81/easyindex"
)

const (
	PublishAPIDefaultQuota = 200
)

type (
	// PublishRequest is defined indexing API request
	PublishRequest struct {
		URL              string
		NotificationType NotificationType
	}

	// SkipedPublishRequest is defined skiped indexing API request
	SkipedPublishRequest struct {
		URL              string
		NotificationType NotificationType
		Reason           error
	}

	// Publisher ...
	Publisher interface {
		Precheck(input []PublishRequest) (requests []PublishRequest, skips []SkipedPublishRequest)
		Publish(request PublishRequest) (responses *indexing.PublishUrlNotificationResponse, skip bool, err error)
	}

	publishImpl struct {
		// Quota is Publish API requests limit per day
		quota int

		// client sends Google indexing API request
		client APIClient

		// Crawler is the instance of the client accessing your web page
		crawler *http.Client

		// logger is logger instance
		logger *zap.Logger

		// Logging is the flag if outputs logs
		logging bool
	}

	Config struct {
		// Quota is Publish API requests limit per day
		Quota *int

		// Context is used to send Google indexing API request
		Context context.Context

		// CredentialsFile is Google APIs credentials file path.
		//   You must set either CredentialsFile or Token.
		//   If both are set, Token will be given priority.
		CredentialsFile *string

		// CredentialsFile is Google APIs credentials file path.
		//   You must set either CredentialsFile or Token.
		//   If both are set, Token will be given priority.
		Token oauth2.TokenSource

		// Crawler is the instance of the client accessing your web page
		Crawler *http.Client

		// Logger is logger instance
		Logger *zap.Logger

		// Logging is the flag if outputs logs
		Logging bool
	}
)

func MustPublish(conf Config) Publisher {
	p, err := Publish(conf)
	if err != nil {
		panic((err))
	}
	return p
}

func Publish(conf Config) (Publisher, error) {
	ctx := conf.Context
	if ctx == nil {
		ctx = context.Background()
	}
	var c APIClient
	var err error
	if conf.Token != nil {
		if c, err = NewByWithToken(ctx, conf.Token); err != nil {
			return nil, err
		}
	} else if conf.CredentialsFile != nil {
		if c, err = NewByWithCredentialsFile(ctx, *conf.CredentialsFile); err != nil {
			return nil, err
		}
	} else {
		err = errors.New("API credential is not given")
		return nil, err
	}
	return PublishWithClient(conf, c), nil
}

// New creates new coordinator instance with API client
func PublishWithClient(conf Config, c APIClient) Publisher {
	crawler := conf.Crawler
	if crawler == nil {
		crawler = http.DefaultClient
	}
	quota := PublishAPIDefaultQuota
	if conf.Quota != nil && *conf.Quota >= 0 {
		quota = *conf.Quota
	}

	return &publishImpl{
		quota:   quota,
		client:  c,
		crawler: crawler,
		logger:  conf.Logger,
		logging: conf.Logging,
	}
}

func (i *publishImpl) Precheck(rs []PublishRequest) (requests []PublishRequest, skips []SkipedPublishRequest) {
	skips = []SkipedPublishRequest{}
	requests = []PublishRequest{}

	remain := i.quota
	for _, r := range rs {
		if remain <= 0 {
			skips = append(skips, SkipedPublishRequest{
				NotificationType: r.NotificationType,
				URL:              r.URL,
				Reason:           easyindex.ErrExceededQuota,
			})
		} else if r.NotificationType == NotificationTypeUpdated {
			resp, err := i.crawler.Get(r.URL)
			if err != nil {
				skips = append(skips, SkipedPublishRequest{
					NotificationType: r.NotificationType,
					URL:              r.URL,
					Reason:           err,
				})
			} else if resp.StatusCode > 300 {
				// error is returned　If status code is not 2xx. Includes redirects.
				skips = append(skips, SkipedPublishRequest{
					NotificationType: r.NotificationType,
					URL:              r.URL,
					Reason:           errors.New(resp.Status),
				})
			} else {
				requests = append(requests, r)
			}
			remain--
		} else {
			requests = append(requests, r)
			remain--
		}
	}
	return
}

func (i *publishImpl) Publish(r PublishRequest) (responses *indexing.PublishUrlNotificationResponse, skip bool, err error) {
	if i.quota <= 0 {
		return nil, true, easyindex.ErrExceededQuota
	}

	return i.request(r.URL, r.NotificationType, false)
}

func (i *publishImpl) request(u string, nt NotificationType, isRetried bool) (resp *indexing.PublishUrlNotificationResponse, skip bool, err error) {
	if i.quota <= 0 {
		return nil, true, easyindex.ErrExceededQuota
	}
	i.quota--
	if resp, err = i.client.Publish(u, nt); err != nil {
		i.error("publish.request", zap.String("url", u), zap.String("type", string(nt)), zap.Error(err))
		if resp != nil {
			if resp.HTTPStatusCode == http.StatusTooManyRequests {
				return resp, false, easyindex.ErrExceededQuota
			}
			if !isRetried && (resp.HTTPStatusCode == http.StatusBadGateway || resp.HTTPStatusCode == http.StatusServiceUnavailable) {
				r, _, e := i.request(u, nt, true)
				if e != nil || r.HTTPStatusCode >= 400 {
					i.error("publish.request:retry", zap.String("url", u), zap.String("type", string(nt)), zap.Error(err))
					return resp, false, err
				}
			}
		}
	}
	return
}

func (i *publishImpl) error(msg string, fields ...zapcore.Field) {
	if i.logging && i.logger != nil {
		i.logger.Error(msg, fields...)
	}
}
