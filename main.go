package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
)

func main() {
	if err := mainError(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

type notification struct {
	Id         string              `json:"id"`
	Repository notificationRepo    `json:"repository"`
	Subject    notificationSubject `json:"subject"`
	Reason     string              `json:"reason"`
	Unread     bool                `json:"unread"`
	UpdatedAt  string              `json:"updated_at"`
	ThreadUrl  string              `json:"url"`
}

type notificationRepo struct {
	FullName string `json:"full_name"`
}

type notificationSubject struct {
	Type  string `json:"type"`
	Url   string `json:"url"`
	Title string `json:"title"`
}

type abstractNotificationSubjectState struct {
	State string `json:"state"`
	Title string `json:"title"`
}

const DefaultN = 100

type customTripper struct {
	inner http.RoundTripper
}

func (c *customTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	resp, err := c.inner.RoundTrip(request)
	if resp != nil {
		oldBody := resp.Body
		defer oldBody.Close()
		buff, err := io.ReadAll(oldBody)
		if err != nil {
			return resp, err
		}
		if len(buff) == 0 {
			resp.Body = io.NopCloser(bytes.NewReader([]byte("null")))
		} else {
			resp.Body = io.NopCloser(bytes.NewReader(buff))
		}
	}
	return resp, err
}

func shouldMarkNotificationAsRead(client *api.RESTClient, n notification, log *slog.Logger) (bool, error) {
	u, err := url.Parse(n.Subject.Url)
	if err != nil {
		return false, fmt.Errorf("failed to parse subject url for notification: '%s': %w", n.Subject.Url, err)
	}
	if u.Path == "" {
		log.Info("not marking repo level notification as read", slog.Any("notification", n))
		return false, nil
	}
	u.Path = strings.TrimPrefix(u.Path, "/")
	var stateResp abstractNotificationSubjectState
	log.Info("looking up notification subject", slog.String("subject_path", u.Path))
	if err := client.Get(u.Path, &stateResp); err != nil {
		return false, fmt.Errorf("failed to lookup notification subject: %w", err)
	}
	log.Debug("got notification subject", slog.Any("subject", stateResp))

	return stateResp.State == "closed", nil
}

func mainError() error {
	if v := os.Getenv("GH_CLEANUP_NOTIFICATIONS_DEBUG"); v != "" {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true})))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	}
	client, err := api.NewRESTClient(api.ClientOptions{
		Headers:   map[string]string{"Accept": "application/vnd.github+json"},
		Transport: &customTripper{inner: http.DefaultTransport},
	})
	if err != nil {
		return fmt.Errorf("failed to setup client: %w", err)
	}

	var before string
	var seen int
	for {
		slog.Info("retrieving page of notifications", slog.Int("n", DefaultN), slog.String("before", before))
		var resp []notification
		if err := client.Get(fmt.Sprintf("notifications?per_page=%d&before=%s", DefaultN, url.QueryEscape(before)), &resp); err != nil {
			return fmt.Errorf("failed to list notifications: %w", err)
		}
		slog.Debug("got notifications", slog.Int("n", len(resp)), slog.Any("notifications", resp))
		for _, n := range resp {
			seen += 1
			if mark, err := shouldMarkNotificationAsRead(client, n, slog.Default().With(slog.Int("seen", seen))); err != nil {
				return err
			} else if mark {
				threadUrl, err := url.Parse(n.ThreadUrl)
				if err != nil {
					return fmt.Errorf("failed to parse thread url '%s': %w", n.ThreadUrl, err)
				}
				threadUrl.Path = strings.TrimPrefix(threadUrl.Path, "/")
				slog.Info("marking notification as read", slog.String("thread_path", threadUrl.Path), slog.Int("seen", seen))
				var out json.RawMessage
				if err := client.Patch(threadUrl.Path, nil, &out); err != nil {
					return fmt.Errorf("failed to mark as read: %w", err)
				}
			}
		}
		if len(resp) == 0 {
			break
		}
		before = resp[len(resp)-1].UpdatedAt
	}

	return nil
}
