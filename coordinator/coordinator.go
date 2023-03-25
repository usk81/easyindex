package coordinator

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/usk81/easyindex"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/oauth2"
	"google.golang.org/api/indexing/v3"
)

type (
	// PublishRequest is defined indexing API request
	PublishRequest struct {
		URL              string
		NotificationType easyindex.NotificationType
	}

	// SkipedPublishRequest is defined skiped indexing API request
	SkipedPublishRequest struct {
		URL              string
		NotificationType easyindex.NotificationType
		Reason           error
	}

	Config struct {
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

		// checks if your web page can be accessed before sending the request to Google indexing API if IgnorePreCheck is false
		//   default: false
		IgnorePreCheck bool

		// Skip is set when there is a problem of pre-check.
		//   If true, request the API by excluding web pages that had problems with pre-checking.
		//   If false, it will exit with an error if there is a problem with the pre-check.
		//   default: false
		Skip bool

		// Crawler is the instance of the client accessing your web page
		Crawler *http.Client

		// Logger is logger instance
		Logger *zap.Logger
	}

	Service struct {
		// client sends Google indexing API request
		client easyindex.APIClient

		// checks if your updated web pages can be accessed before sending the request to Google indexing API if IgnorePreCheck is false
		//   default: false
		ignorePreCheck bool

		// Skip is set when there is a problem of pre-check.
		//   If true, request the API by excluding web pages that had problems with pre-checking.
		//   If false, it will exit with an error if there is a problem with the pre-check.
		//   default: false
		skip bool

		// Crawler is the instance of the client accessing your web page
		crawler *http.Client

		// logger is logger instance
		logger *zap.Logger
	}

	// Publisher ...
	Publisher interface {
		Publish(requests []PublishRequest, quota ...int) (
			// Total number of requests
			total int,
			// Count of API request
			count int,
			// Responses from Google Indexing API
			responses []*indexing.PublishUrlNotificationResponse,
			// skiped API Requests
			skips []SkipedPublishRequest,
			err error,
		)
	}

	// Manager...
	Manager interface {
		Publisher
	}
)

var (
	// ExceededQuota ...
	ErrExceededQuota = errors.New("quota exceeded")
)

// New creates new coordinator instance
func New(conf Config) (s Manager, err error) {
	ctx := conf.Context
	if ctx == nil {
		ctx = context.Background()
	}

	var c easyindex.APIClient
	if conf.Token != nil {
		if c, err = easyindex.NewByWithToken(ctx, conf.Token); err != nil {
			return
		}
	} else if conf.CredentialsFile != nil {
		if c, err = easyindex.NewByWithCredentialsFile(ctx, *conf.CredentialsFile); err != nil {
			return
		}
	} else {
		err = errors.New("API credential is not given")
		return
	}
	return NewWithClient(conf, c), nil
}

// New creates new coordinator instance with API client
func NewWithClient(conf Config, c easyindex.APIClient) Manager {
	crawler := conf.Crawler
	if crawler == nil {
		crawler = http.DefaultClient
	}
	return &Service{
		client:         c,
		ignorePreCheck: conf.IgnorePreCheck,
		skip:           conf.Skip,
		crawler:        crawler,
		logger:         conf.Logger,
	}
}

// Publish sends Google indexing API requests
func (s *Service) Publish(requests []PublishRequest, quota ...int) (
	// Total number of requests
	total int,
	// Count of API request
	count int,
	// Responses from Google Indexing API
	responses []*indexing.PublishUrlNotificationResponse,
	// skiped API Requests
	skips []SkipedPublishRequest,
	err error,
) {
	rs := requests
	total = len(rs)
	skips = []SkipedPublishRequest{}
	responses = []*indexing.PublishUrlNotificationResponse{}

	if !s.ignorePreCheck {
		if s.skip {
			rs, skips = s.appendSkips(rs)
		} else {
			if err = s.alertIfError(rs); err != nil {
				return
			}
		}
	}

	q := 0
	if len(quota) > 0 && quota[0] > 0 {
		q = quota[0]
	}
	if q > 0 && len(rs) > q {
		// sets SkipedPublishRequest
		ss := rs[q:]
		for _, v := range ss {
			skips = append(skips, SkipedPublishRequest{
				NotificationType: v.NotificationType,
				URL:              v.URL,
				Reason:           ErrExceededQuota,
			})
		}

		// removs unused requests
		rs = rs[:q]
	}

	for _, r := range rs {
		count++
		s.debug("publish", zap.String("url", r.URL), zap.String("type", string(r.NotificationType)))
		var resp *indexing.PublishUrlNotificationResponse
		if resp, err = s.request(r.URL, r.NotificationType, false); err != nil {
			if resp != nil {
				return
			}
			s.error("publish", zap.String("url", r.URL), zap.String("type", string(r.NotificationType)), zap.Error(err))
			return
		}
		responses = append(responses, resp)
	}
	return
}

func (s *Service) request(u string, nt easyindex.NotificationType, isRetried bool) (resp *indexing.PublishUrlNotificationResponse, err error) {
	if resp, err = s.client.Publish(u, nt); err != nil {
		s.error("publish", zap.String("url", u), zap.String("type", string(nt)), zap.Error(err))
		if !isRetried && resp != nil && (resp.HTTPStatusCode == http.StatusBadGateway || resp.HTTPStatusCode == http.StatusServiceUnavailable) {
			r, e := s.request(u, nt, true)
			if e != nil || r.HTTPStatusCode >= 400 {
				s.error("publish:retry", zap.String("url", u), zap.String("type", string(nt)), zap.Error(err))
				return resp, err
			}
		}
	}
	return
}

func (s *Service) debug(msg string, fields ...zapcore.Field) {
	if s.logger != nil {
		s.logger.Debug(msg, fields...)
	}
}

func (s *Service) error(msg string, fields ...zapcore.Field) {
	if s.logger != nil {
		s.logger.Error(msg, fields...)
	}
}

func (s *Service) alertIfError(rs []PublishRequest) (err error) {
	for _, r := range rs {
		if r.NotificationType == easyindex.NotificationTypeUpdated {
			var resp *http.Response
			if resp, err = s.crawler.Get(r.URL); err != nil {
				s.error("pre-check", zap.String("url", r.URL), zap.String("type", string(r.NotificationType)), zap.Error(err))
				return
			}
			// error is returned　If status code is not 2xx. Includes redirects.
			if resp.StatusCode > 300 {
				err = fmt.Errorf("pre-check : %s : %s", resp.Status, r.URL)
				s.error("pre-check", zap.String("url", r.URL), zap.String("type", string(r.NotificationType)), zap.Error(err))
				return
			}
		}
	}
	return nil
}

func (s *Service) appendSkips(rs []PublishRequest) (requests []PublishRequest, skips []SkipedPublishRequest) {
	skips = []SkipedPublishRequest{}
	requests = []PublishRequest{}

	for _, r := range rs {
		if r.NotificationType == easyindex.NotificationTypeUpdated {
			resp, err := s.crawler.Get(r.URL)
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
					Reason:           fmt.Errorf("%s : %s", resp.Status, r.URL),
				})
			} else {
				requests = append(requests, r)
			}
		} else {
			requests = append(requests, r)
		}
	}
	return
}
