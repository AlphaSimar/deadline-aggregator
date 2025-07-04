package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type DiscordWebhook struct {
	Content string        `json:"content,omitempty"`
	Embeds  []DiscordEmbed `json:"embeds,omitempty"`
}

type DiscordEmbed struct {
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Color       int                    `json:"color,omitempty"`
	Fields      []DiscordEmbedField    `json:"fields,omitempty"`
	Timestamp   string                 `json:"timestamp,omitempty"`
	Footer      *DiscordEmbedFooter    `json:"footer,omitempty"`
}

type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type DiscordEmbedFooter struct {
	Text string `json:"text"`
}

func SendAssignmentReminder(assignments []AssignmentInfo) error {
	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	if webhookURL == "" {
		return fmt.Errorf("DISCORD_WEBHOOK_URL environment variable not set")
	}

	if len(assignments) == 0 {
		return nil
	}

	embed := DiscordEmbed{
		Title:       "⚠️ Assignment Reminders",
		Description: fmt.Sprintf("You have **%d** assignment(s) due within 6 hours!", len(assignments)),
		Color:       16776960, 
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &DiscordEmbedFooter{
			Text: "Deadline Aggregator",
		},
	}

	for _, assignment := range assignments {
		timeLeft := assignment.DueTime.Sub(time.Now())
		hours := int(timeLeft.Hours())
		minutes := int(timeLeft.Minutes()) % 60

		field := DiscordEmbedField{
			Name:   assignment.Title,
			Value:  fmt.Sprintf("**Course:** %s\n**Due:** %s\n**Time Left:** %dh %dm", assignment.CourseName, assignment.DueTime.Format("Jan 2, 2006 at 3:04 PM"), hours, minutes),
			Inline: false,
		}
		embed.Fields = append(embed.Fields, field)
	}

	webhook := DiscordWebhook{
		Embeds: []DiscordEmbed{embed},
	}

	jsonData, err := json.Marshal(webhook)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook: %v", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("discord webhook returned status: %d", resp.StatusCode)
	}

	log.Printf("Sent Discord notification for %d assignments", len(assignments))
	return nil
}

type AssignmentInfo struct {
	Title      string
	CourseName string
	DueTime    time.Time
	CourseID   string
} 