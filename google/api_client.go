package google

import (
	"context"

	"golang.org/x/oauth2"
	"google.golang.org/api/indexing/v3"
	"google.golang.org/api/option"
)

// APIClient is interface of client to request Google Indexing API
type APIClient interface {
	Publish(u string, nt NotificationType) (result *indexing.PublishUrlNotificationResponse, err error)
}

// Client is defined parameters to request Google Indexing API
type Client struct {
	Service *indexing.Service
}

// New creates new instance to request Google Indexing API
func New(ctx context.Context, authOption option.ClientOption) (c APIClient, err error) {
	s, err := indexing.NewService(ctx, authOption)
	if err != nil {
		return
	}
	return &Client{
		Service: s,
	}, nil
}

func NewByWithCredentialsFile(ctx context.Context, fp string) (c APIClient, err error) {
	return New(ctx, option.WithCredentialsFile(fp))
}

func NewByWithToken(ctx context.Context, token oauth2.TokenSource) (c APIClient, err error) {
	return New(ctx, option.WithTokenSource(token))
}

// Execute Google Indexing API
func (c *Client) Publish(u string, nt NotificationType) (result *indexing.PublishUrlNotificationResponse, err error) {
	return c.Service.UrlNotifications.Publish(&indexing.UrlNotification{
		Url:  u,
		Type: string(nt),
	}).Do()
}
