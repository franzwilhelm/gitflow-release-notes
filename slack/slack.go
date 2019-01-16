package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/franzwilhelm/gitflow-release-notes/gitflow"
	"github.com/google/go-github/github"
	slackify "github.com/karriereat/blackfriday-slack"
)

var webhookURL string

// WebhookMessage holds the message to send to slack
type WebhookMessage struct {
	Channel     string       `json:"channel"`
	Username    string       `json:"username,omitempty"`
	IconURL     string       `json:"icon_url,omitempty"`
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Attachment contains the information needed for release note attachments
type Attachment struct {
	Color      string   `json:"color,omitempty"`
	Title      string   `json:"title,omitempty"`
	TitleLink  string   `json:"title_link,omitempty"`
	Pretext    string   `json:"pretext,omitempty"`
	Text       string   `json:"text"`
	ImageURL   string   `json:"image_url,omitempty"`
	ThumbURL   string   `json:"thumb_url,omitempty"`
	MarkdownIn []string `json:"mrkdwn_in,omitempty"`
	Footer     string   `json:"footer,omitempty"`
	FooterIcon string   `json:"footer_icon,omitempty"`
}

// UsePullRequests formats the data from pull requests and adds them to the attachment
func (a *Attachment) UsePullRequests(prs []github.PullRequest) {
	a.Text += "──────\n"
	for _, pr := range prs {
		title := fmt.Sprintf("<%s|#%v>: _*%s*_", pr.GetHTMLURL(), pr.GetNumber(), gitflow.RemovePrefixes(pr.GetTitle()))
		a.Text += fmt.Sprintf("%s\n", title)
		if pr.GetBody() != "" {
			body := string(slackify.Run([]byte(pr.GetBody())))
			a.Text += fmt.Sprintf("%s\n", strings.TrimRight(body, "\n"))
		}
		a.Text += "\n"
	}
	a.MarkdownIn = []string{"text"}
}

// Initialize sets the webhook url of the slack request
func Initialize(url string) {
	webhookURL = url
}

// PostWebhook posts a WebhookMessage to slack
func PostWebhook(msg *WebhookMessage) error {
	raw, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	response, err := http.Post(webhookURL, "application/json", bytes.NewReader(raw))

	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Got status code %v", response.StatusCode)
	}

	return nil
}
