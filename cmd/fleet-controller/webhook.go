package main

import (
	"context"
	"fmt"

	"github.com/mattermost/fleet-controller/internal/webhook"
)

func sendWebhook(webhookURL, text string) error {
	ctx := context.TODO()

	payload := &webhook.Payload{
		Username: "Fleet Controller",
		IconURL:  "https://static.wikia.nocookie.net/starwars/images/a/a7/ISD_arrow.jpg/revision/latest/scale-to-width-down/870?cb=20070424053722",
		Text:     text,
	}

	return webhook.Send(ctx, webhookURL, payload)
}

const errorWebhookMessage = `### Fleet Controller Encountered an Error

Run ID: %s

Error: %s
`

func sendErrorWebhook(webhookURL, runID string, err error) {
	if len(webhookURL) == 0 {
		return
	}

	sendWebhook(webhookURL, fmt.Sprintf(errorWebhookMessage, wrapInlineCode(runID), wrapCodeBlock(err.Error())))
}

const hibernateReportMessage = `### Hibernation Report

Run ID: %s
Runtime: %s
Filters:
 - Days: %d
 - Max Users: %d
 - Group ID: %s
 - Owner ID: %s

#### Results
| Type | Count |
| -- | -- |
| Original Stable Installations | %d | 
| Installations Hibernated | %d |
| Installations Skipped (User Count) | %d |
| Hibernation Calculation Errors | %d |
`

func sendHibernateWebhook(webhookURL, runID, runtime, groupID, ownerID string, days, maxUsers, stableCount, hibernatedCount, skippedCount, errorCount int) error {
	webhookText := fmt.Sprintf(
		hibernateReportMessage,         // Text template
		wrapInlineCode(runID), runtime, // Run data
		days, maxUsers, wrapInlineCode(groupID), wrapInlineCode(ownerID), // Filters
		stableCount, hibernatedCount, skippedCount, errorCount, // Results
	)

	return sendWebhook(webhookURL, webhookText)
}

func wrapInlineCode(s string) string {
	return fmt.Sprintf("`%s`", s)
}

func wrapCodeBlock(s string) string {
	return fmt.Sprintf("\n```\n%s\n```\n", s)
}
