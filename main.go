package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/mailgun/mailgun-go"
	"gorm.io/driver/postgres"

	"gorm.io/gorm"
)

type User struct {
	ID     uint   `gorm:"primaryKey"`
	Email  string `gorm:"unique"`
	Status string // "waiting" or "matched"
}

type Match struct {
	ID       uint `gorm:"primaryKey"`
	User1    string
	User2    string
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

	// Check if user already exists
	var existingUser User
	db.Where("email = ?", email).First(&existingUser)
	if existingUser.ID != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email already registered"})
		return
	}

	// Register new user
	newUser := User{Email: email, Status: "waiting"}
	db.Create(&newUser)

	// Check for matching users
	var waitingUser User
	db.Where("status = ?", "waiting").Not("email = ?", email).First(&waitingUser)

	if waitingUser.ID != 0 {
		// Match found, create Google Meet link
		meetLink := "https://meet.google.com/random-meeting-id"

		// Save match
		match := Match{User1: waitingUser.Email, User2: email, MeetLink: meetLink}
		db.Create(&match)

		// Update statuses
		db.Model(&waitingUser).Update("status", "matched")
		db.Model(&newUser).Update("status", "matched")

		// Send emails
		sendEmail(waitingUser.Email, meetLink)
		sendEmail(email, meetLink)

		c.JSON(http.StatusOK, gin.H{"message": "Matched!", "meet_link": meetLink})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "Waiting for a match..."})
	}
}

func sendEmail(to string, meetLink string) {
	domain := "mock.interview.com" // e.g., "sandbox123.mailgun.org"
	apiKey := "75d948e91a529329d6f2f93617411118-3d4b3a2a-ddac9672"

	// Sender & recipient details
	sender := "Mock Interview <mock.interview.com>" // Must be from your verified domain
	recipient := to                                 // Change to the actual recipient email
	subject := "Mock Interview Scheduled!"
	body := "Your mock interview has been scheduled. Check your Google Meet link!"

	// Initialize Mailgun client
	mg := mailgun.NewMailgun(domain, apiKey)

	// Create a message
	message := mg.NewMessage(sender, subject, body, recipient)

	// Set a timeout context
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	// defer cancel()

	// Send the email
	_, _, err := mg.Send(message)
	if err != nil {

	}

	fmt.Println("Email sent successfully!")
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
