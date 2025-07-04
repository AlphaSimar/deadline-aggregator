package scheduler

import (
	"database/sql"
	"log"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"deadline-aggregator/internal/discord"
	classroomapi "deadline-aggregator/internal/google"
	"deadline-aggregator/internal/store"
)

func StartReminderScheduler(db *sql.DB) {
	log.Println("Starting reminder scheduler...")
	
	oauthConfig := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/classroom.courses.readonly",
			"https://www.googleapis.com/auth/classroom.coursework.me.readonly",
			"openid",
		},
		Endpoint: google.Endpoint,
	}
	
	for {
		now := time.Now()
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, now.Location())
		if !nextRun.After(now) {
			nextRun = nextRun.Add(24 * time.Hour)
		}
		sleepDuration := nextRun.Sub(now)
		log.Printf("Scheduler sleeping until %s", nextRun.Format(time.RFC1123))
		time.Sleep(sleepDuration)
		
		log.Println("Scheduler tick - daily check for upcoming assignments...")
		
		userTokens, err := store.GetAllUserTokens(db)
		if err != nil {
			log.Printf("Failed to get user tokens: %v", err)
			continue
		}
		
		if len(userTokens) == 0 {
			log.Println("No users found in database")
			continue
		}
		
		for googleID, token := range userTokens {
			if token.Expiry.Before(time.Now()) {
				log.Printf("Token expired for user %s, attempting to refresh...", googleID)
				continue
			}
			
			assignments, err := classroomapi.GetUpcomingAssignments(token, oauthConfig, 6)
			if err != nil {
				log.Printf("Failed to get assignments for user %s: %v", googleID, err)
				continue
			}
			
			if len(assignments) > 0 {
				log.Printf("Found %d upcoming assignments for user %s:", len(assignments), googleID)
				for _, a := range assignments {
					log.Printf("  - %s | Course: %s | Due: %s", a.Title, a.CourseName, a.DueTime.Format(time.RFC1123))
				}
				
				err = discord.SendAssignmentReminder(assignments)
				if err != nil {
					log.Printf("Failed to send Discord notification for user %s: %v", googleID, err)
					continue
				}
				
				for _, assignment := range assignments {
					log.Printf("Sent notification for assignment: %s (Course: %s)", assignment.Title, assignment.CourseName)
				}
			} else {
				log.Printf("No upcoming assignments within 6h for user %s", googleID)
			}
		}
	}
} 