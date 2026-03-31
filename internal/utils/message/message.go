package message

import (
	"context"
	"fmt"
	"time"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"github.com/slack-go/slack"
)

type MessageHelper interface {
	SendReportMessage(ctx context.Context, mediaType, url string, ip string, count int64, report models.ContentReport) error
}
type SendSlackHelper struct {
	slackClient *slack.Client
	channelID   string
}

func NewSlackHelper(token, channelID string) (MessageHelper, error) {
	slackClient := slack.New(token)
	_, err := slackClient.AuthTest()
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with Slack: %v", err)
	}
	return &SendSlackHelper{
		slackClient: slackClient,
		channelID:   channelID,
	}, nil
}

func (s *SendSlackHelper) SendReportMessage(ctx context.Context, mediaType, url, ip string, count int64, report models.ContentReport) error {
	currentTime := time.Now().Format(time.RFC3339)
	headerBlock := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", ":triangular_flag_on_post: *New Content Report*", false, false),
		nil, nil,
	)

	attachment := slack.Attachment{
		Color: "#FFA500",
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				slack.NewSectionBlock(
					nil,
					[]*slack.TextBlockObject{
						slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*Media ID:*\n`%s`", report.MediaId.String()), false, false),
						slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*Media URL:*\n`%s`", url), false, false),
						slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*IP:*\n`%s`", ip), false, false),
						slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*Media Report Count:*\n`%d`", count), false, false),
						slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*Type:*\n`%s`", mediaType), false, false),
						slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*Reason:*\n`%s`", report.Reason), false, false),
						slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*Created At:*\n`%s`", currentTime), false, false),
					},
					nil,
				),
				slack.NewDividerBlock(),
				slack.NewSectionBlock(
					slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*Description:*\n```%s```", report.Description), false, false),
					nil, nil,
				),
			},
		},
	}

	_, _, err := s.slackClient.PostMessageContext(
		ctx,
		s.channelID,
		slack.MsgOptionBlocks(headerBlock),
		slack.MsgOptionAttachments(attachment),
	)

	if err != nil {
		return fmt.Errorf("error sending Slack message: %v", err)
	}

	return nil
}
