package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"websitego/internal/config"
	"websitego/internal/database"
	"websitego/internal/models"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := database.NewMySQL(cfg)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	const (
		adminEmail       = "admin@gmail.com"
		adminName        = "Admin"
		adminPassword    = "admin789"
		userPassword     = "user12345"
		defaultUserTotal = 9999999
		batchSize        = 1000
	)
	userTotal := flag.Int("users", defaultUserTotal, "number of user-role accounts to create")
	flag.Parse()

	adminHash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("failed to hash admin password: %v", err)
	}
	userHash, err := bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("failed to hash user password: %v", err)
	}

	var admin models.User
	if err := db.Where("email = ?", adminEmail).First(&admin).Error; err == nil {
		admin.Name = adminName
		admin.Role = "admin"
		admin.PasswordHash = string(adminHash)
		if err := db.Save(&admin).Error; err != nil {
			log.Fatalf("failed to update admin user: %v", err)
		}
		log.Printf("admin updated: %s / %s", adminEmail, adminPassword)
	} else {
		admin = models.User{
			Name:         adminName,
			Email:        adminEmail,
			PasswordHash: string(adminHash),
			Role:         "admin",
		}
		if err := db.Create(&admin).Error; err != nil {
			log.Fatalf("failed to create admin user: %v", err)
		}
		log.Printf("admin created: %s / %s", adminEmail, adminPassword)
	}

	if *userTotal > 0 {
		runID := time.Now().Unix()
		created := 0
		users := make([]models.User, 0, batchSize)
		for i := 1; i <= *userTotal; i++ {
			users = append(users, models.User{
				Name:         fmt.Sprintf("User %07d", i),
				Email:        fmt.Sprintf("seed%d_user%07d@gmail.com", runID, i),
				PasswordHash: string(userHash),
				Role:         "user",
			})

			if len(users) == batchSize || i == *userTotal {
				if err := db.Create(&users).Error; err != nil {
					log.Fatalf("failed to create users at batch ending %d: %v", i, err)
				}
				created += len(users)
				users = users[:0]
			}
		}
		log.Printf("seed completed: %d users created", created)
		log.Printf("default user password: %s", userPassword)
		return
	}
	log.Printf("admin reset completed without creating users")
}
