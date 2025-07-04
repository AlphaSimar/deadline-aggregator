package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	googleapi "google.golang.org/api/oauth2/v2"
	classroomapi "deadline-aggregator/internal/google"
	"deadline-aggregator/internal/store"
)

var (
	oauthConfig *oauth2.Config
)

func SetupRoutes(r *gin.Engine, db *sql.DB) {
	oauthConfig = &oauth2.Config{
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

	api := r.Group("/api/v1")
	{
		api.GET("/assignments", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "Assignments will be listed here"})
		})
	}

	auth := r.Group("/auth/google")
	{
		auth.GET("/login", handleGoogleLogin)
		auth.GET("/callback", func(c *gin.Context) {
			handleGoogleCallback(c, db)
		})
	}
}

func handleGoogleLogin(c *gin.Context) {
	url := oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func handleGoogleCallback(c *gin.Context, db *sql.DB) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No code in request"})
		return
	}

	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token exchange failed"})
		return
	}

	client := oauthConfig.Client(context.Background(), token)

	userInfoService, err := googleapi.New(client)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create userinfo client"})
		return
	}

	user, err := userInfoService.Userinfo.V2.Me.Get().Do()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}

	err = store.SaveUserToken(db, user.Id, user.Email, user.Name, token)
	if err != nil {
		log.Printf("Failed to save user token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save user data"})
		return
	}

	log.Printf("User authenticated: %s, Email: %s, ID: %s", user.Name, user.Email, user.Id)

	assignments, err := classroomapi.FetchAssignments(token, oauthConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch assignments"})
		return
	}

	type SimpleAssignment struct {
		Title    string `json:"title"`
		DueDate  string `json:"due_date"`
		CourseId string `json:"course_id"`
	}

	var response []SimpleAssignment
	for _, a := range assignments {
		dueStr := fmt.Sprintf("%d-%02d-%02d %02d:%02d",
			a.DueDate.Year, a.DueDate.Month, a.DueDate.Day,
			a.DueTime.Hours, a.DueTime.Minutes)
		response = append(response, SimpleAssignment{
			Title:    a.Title,
			DueDate:  dueStr,
			CourseId: a.CourseId,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful!",
		"name":    user.Name,
		"email":   user.Email,
		"assignments": response,
	})
}
