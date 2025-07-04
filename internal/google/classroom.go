package googleapi

import (
	"context"
	"log"
	"time"

	classroom "google.golang.org/api/classroom/v1"
	"golang.org/x/oauth2"
	"deadline-aggregator/internal/discord"
)

type AssignmentWithCourse struct {
	Assignment *classroom.CourseWork
	CourseName string
	CourseID   string
}

func FetchAssignments(token *oauth2.Token, config *oauth2.Config) ([]*classroom.CourseWork, error) {
	ctx := context.Background()
	client := config.Client(ctx, token)

	srv, err := classroom.New(client)
	if err != nil {
		return nil, err
	}

	coursesList, err := srv.Courses.List().CourseStates("ACTIVE").Do()
	if err != nil {
		return nil, err
	}

	var allAssignments []*classroom.CourseWork

	for _, course := range coursesList.Courses {
		work, err := srv.Courses.CourseWork.List(course.Id).Do()
		if err != nil {
			log.Printf("Failed to get coursework for course %s: %v", course.Name, err)
			continue
		}

		for _, item := range work.CourseWork {
			if item.DueDate != nil && item.DueTime != nil {
				allAssignments = append(allAssignments, item)
			}
		}
	}

	return allAssignments, nil
}

func FetchAssignmentsWithCourses(token *oauth2.Token, config *oauth2.Config) ([]AssignmentWithCourse, error) {
	ctx := context.Background()
	client := config.Client(ctx, token)

	srv, err := classroom.New(client)
	if err != nil {
		return nil, err
	}

	coursesList, err := srv.Courses.List().CourseStates("ACTIVE").Do()
	if err != nil {
		return nil, err
	}

	var assignmentsWithCourses []AssignmentWithCourse

	for _, course := range coursesList.Courses {
		work, err := srv.Courses.CourseWork.List(course.Id).Do()
		if err != nil {
			log.Printf("Failed to get coursework for course %s: %v", course.Name, err)
			continue
		}

		for _, item := range work.CourseWork {
			if item.DueDate != nil && item.DueTime != nil {
				assignmentsWithCourses = append(assignmentsWithCourses, AssignmentWithCourse{
					Assignment: item,
					CourseName: course.Name,
					CourseID:   course.Id,
				})
			}
		}
	}

	return assignmentsWithCourses, nil
}

func GetUpcomingAssignments(token *oauth2.Token, config *oauth2.Config, hours int) ([]discord.AssignmentInfo, error) {
	assignments, err := FetchAssignmentsWithCourses(token, config)
	if err != nil {
		return nil, err
	}

	var upcomingAssignments []discord.AssignmentInfo
	now := time.Now()
	deadline := now.Add(time.Duration(hours) * time.Hour)

	for _, assignment := range assignments {
		utcTime := time.Date(
			int(assignment.Assignment.DueDate.Year),
			time.Month(assignment.Assignment.DueDate.Month),
			int(assignment.Assignment.DueDate.Day),
			int(assignment.Assignment.DueTime.Hours),
			int(assignment.Assignment.DueTime.Minutes),
			0, 0, time.UTC,
		)
		dueTime := utcTime.In(time.Local)

		if dueTime.After(now) && dueTime.Before(deadline) {
			upcomingAssignments = append(upcomingAssignments, discord.AssignmentInfo{
				Title:      assignment.Assignment.Title,
				CourseName: assignment.CourseName,
				DueTime:    dueTime,
				CourseID:   assignment.CourseID,
			})
		}
	}

	return upcomingAssignments, nil
}
