package store

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/oauth2"
)

func NewPostgres() (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSL"),
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	return db, db.Ping()
}

func RunMigrations(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			google_id VARCHAR(255) UNIQUE NOT NULL,
			email VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			access_token TEXT,
			refresh_token TEXT,
			token_expiry TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create users table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS assignments (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id),
			course_id VARCHAR(255) NOT NULL,
			course_name VARCHAR(255) NOT NULL,
			assignment_id VARCHAR(255) NOT NULL,
			title VARCHAR(500) NOT NULL,
			due_date TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, assignment_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create assignments table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS notifications (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id),
			assignment_id VARCHAR(255) NOT NULL,
			sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			notification_type VARCHAR(50) NOT NULL,
			UNIQUE(user_id, assignment_id, notification_type, sent_at)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create notifications table: %v", err)
	}

	return nil
}

func SaveUserToken(db *sql.DB, googleID, email, name string, token *oauth2.Token) error {
	_, err := db.Exec(`
		INSERT INTO users (google_id, email, name, access_token, refresh_token, token_expiry, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)
		ON CONFLICT (google_id) 
		DO UPDATE SET 
			email = $2, 
			name = $3, 
			access_token = $4, 
			refresh_token = $5, 
			token_expiry = $6, 
			updated_at = CURRENT_TIMESTAMP
	`, googleID, email, name, token.AccessToken, token.RefreshToken, token.Expiry)
	
	return err
}

func GetUserToken(db *sql.DB, googleID string) (*oauth2.Token, error) {
	var accessToken, refreshToken string
	var expiry time.Time

	err := db.QueryRow(`
		SELECT access_token, refresh_token, token_expiry 
		FROM users 
		WHERE google_id = $1
	`, googleID).Scan(&accessToken, &refreshToken, &expiry)
	
	if err != nil {
		return nil, err
	}

	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       expiry,
	}

	return token, nil
}

func GetAllUserTokens(db *sql.DB) (map[string]*oauth2.Token, error) {
	rows, err := db.Query(`
		SELECT google_id, access_token, refresh_token, token_expiry 
		FROM users 
		WHERE access_token IS NOT NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := make(map[string]*oauth2.Token)
	for rows.Next() {
		var googleID, accessToken, refreshToken string
		var expiry time.Time

		err := rows.Scan(&googleID, &accessToken, &refreshToken, &expiry)
		if err != nil {
			continue
		}

		tokens[googleID] = &oauth2.Token{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			Expiry:       expiry,
		}
	}

	return tokens, nil
}

func SaveNotificationSent(db *sql.DB, userID int, assignmentID, notificationType string) error {
	_, err := db.Exec(`
		INSERT INTO notifications (user_id, assignment_id, notification_type)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, assignment_id, notification_type, DATE(sent_at)) DO NOTHING
	`, userID, assignmentID, notificationType)
	
	return err
}

func HasNotificationBeenSent(db *sql.DB, userID int, assignmentID, notificationType string) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) 
		FROM notifications 
		WHERE user_id = $1 AND assignment_id = $2 AND notification_type = $3 
		AND DATE(sent_at) = CURRENT_DATE
	`, userID, assignmentID, notificationType).Scan(&count)
	
	return count > 0, err
}
