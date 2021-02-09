package main

import (
	"context"
	"fmt"

	"github.com/mattermost/fleet-controller/internal/webhook"
)

const hibernateReportMessage = `### Hibernation Report

Run ID: %s
Runtime: %s

| Original Stable Installation Count | Installations Hibernated | Installations Skipped (User Count) | Hibernation Calculation Errors |
| -- | -- | -- | -- |
| %d | %d | %d | %d |
`

func sendHibernateReportWebhook(webhookURL, text string) error {
	ctx := context.TODO()

	payload := &webhook.Payload{
		Username: "Fleet Controller",
		IconURL:  "https://static.wikia.nocookie.net/starwars/images/a/a7/ISD_arrow.jpg/revision/latest/scale-to-width-down/870?cb=20070424053722",
		Text:     text,
	}

	return webhook.Send(ctx, webhookURL, payload)
}

func wrapInlineCode(s string) string {
	return fmt.Sprintf("`%s`", s)
}
