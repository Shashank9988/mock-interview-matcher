package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/resendlabs/resend-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	ID       uint   `gorm:"primaryKey"`
	Email    string `gorm:"unique"`
	TimeSlot string // New field to store the selected time slot
	Status   string // "waiting" or "matched"
}

type Match struct {
	ID       uint `gorm:"primaryKey"`
	User1    string
	User2    string
	TimeSlot string // Ensure users are matched by time slot
	MeetLink string
}

var db *gorm.DB

func initDB() {
	dsn := "postgresql://neondb_owner:npg_kATYwzj1GSb3@ep-royal-snow-a5b9w63w-pooler.us-east-2.aws.neon.tech/neondb?sslmode=require"
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	db.AutoMigrate(&User{}, &Match{})
}

func registerUser(c *gin.Context) {
	email := c.PostForm("email")
	timeSlot := c.PostForm("time_slot") // Get the time slot from the form data

	if email == "" || timeSlot == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email and time_slot are required"})
		return
	}

	// Check if user already exists with the same time slot
	var existingUser User
	db.Where("email = ?", email).First(&existingUser)
	if existingUser.ID != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email already registered"})
		return
	}

	// Register new user
	newUser := User{Email: email, TimeSlot: timeSlot, Status: "waiting"}
	db.Create(&newUser)

	// Check for a matching user in the same time slot
	var waitingUser User
	db.Where("status = ? AND time_slot = ?", "waiting", timeSlot).Not("email = ?", email).First(&waitingUser)

	if waitingUser.ID != 0 {
		// Match found, create Google Meet link
		meetLink := "https://meet.google.com/mpk-erqa-kqd"

		// Save match
		match := Match{User1: waitingUser.Email, User2: email, TimeSlot: timeSlot, MeetLink: meetLink}
		db.Create(&match)

		// Delete both users entry
		db.Delete(&waitingUser)
		db.Delete(&newUser)

		// Send emails
		sendEmail(waitingUser.Email, meetLink)
		sendEmail(email, meetLink)

		c.JSON(http.StatusOK, gin.H{"message": "Matched!", "meet_link": meetLink})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "Waiting for a match..."})
	}
}

func sendEmail(to string, meetLink string) {
	apiKey := "re_9ZgeUBtA_EvRqt3SDipSvtXcMWBJpuatc"

	client := resend.NewClient(apiKey)

	params := &resend.SendEmailRequest{
		From:    "mock.interview.com", // Must be a verified email/domain
		To:      []string{to},
		Subject: "Mock Interview Scheduled!",
		Text:    fmt.Sprintf("Your mock interview is scheduled at your selected time slot. Join using this link: %s", meetLink),
	}

	// Send the email
	email, err := client.Emails.Send(params)
	if err != nil {
		log.Println("Failed to send email:", err)
		return
	}

	fmt.Printf("Email sent successfully! Email ID: %s\n", email.Id)
}

func main() {
	initDB()

	r := gin.Default()
	r.POST("/register", registerUser)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
