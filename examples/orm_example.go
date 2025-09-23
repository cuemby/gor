package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cuemby/gor/internal/orm"
	"github.com/cuemby/gor/pkg/gor"
)

// Example models
type User struct {
	gor.BaseModel
	Email     string    `gor:"unique;not_null" json:"email"`
	Username  string    `gor:"unique;not_null" json:"username"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Active    bool      `gor:"not_null" json:"active"`
}

func (u User) TableName() string {
	return "users"
}

type Post struct {
	gor.BaseModel
	Title   string `gor:"not_null" json:"title"`
	Content string `json:"content"`
	UserID  uint   `gor:"not_null;index" json:"user_id"`
	Published bool `gor:"not_null" json:"published"`
}

func (p Post) TableName() string {
	return "posts"
}

func main() {
	// Create ORM instance
	config := gor.DatabaseConfig{
		Driver:   "sqlite3",
		Database: "./test.db", // File database for this example
	}

	gorORM := orm.NewORM(config)

	// Connect to database
	ctx := context.Background()
	if err := gorORM.Connect(ctx, config); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer gorORM.Close()

	// Register models
	if err := gorORM.Register(&User{}, &Post{}); err != nil {
		log.Fatal("Failed to register models:", err)
	}

	fmt.Println("🎉 Connected to database and registered models")

	// Debug: Test manual table creation
	_, err := gorORM.DB().Exec("CREATE TABLE IF NOT EXISTS test_table (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		log.Fatal("Failed to create test table:", err)
	}

	// Debug: Check if tables exist
	rows, err := gorORM.DB().Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		log.Fatal("Failed to query tables:", err)
	}
	defer rows.Close()

	fmt.Println("📋 Tables in database:")
	for rows.Next() {
		var tableName string
		rows.Scan(&tableName)
		fmt.Printf("  - %s\n", tableName)
	}

	// Create a user
	user := &User{
		Email:     "john@example.com",
		Username:  "johndoe",
		FirstName: "John",
		LastName:  "Doe",
		Active:    true,
	}

	if err := gorORM.Create(user); err != nil {
		log.Fatal("Failed to create user:", err)
	}

	fmt.Printf("✅ Created user: %+v\n", user)

	// Create some posts
	posts := []*Post{
		{
			Title:     "My First Post",
			Content:   "This is my first blog post!",
			UserID:    user.ID,
			Published: true,
		},
		{
			Title:     "Draft Post",
			Content:   "This is a draft post",
			UserID:    user.ID,
			Published: false,
		},
	}

	for _, post := range posts {
		if err := gorORM.Create(post); err != nil {
			log.Fatal("Failed to create post:", err)
		}
	}

	fmt.Printf("✅ Created %d posts\n", len(posts))

	// Query examples
	fmt.Println("\n📊 Query Examples:")

	// Find user by ID
	var foundUser User
	if err := gorORM.Find(&foundUser, user.ID); err != nil {
		log.Fatal("Failed to find user:", err)
	}
	fmt.Printf("Found user by ID: %s (%s)\n", foundUser.Username, foundUser.Email)

	// Find all users
	var users []User
	if err := gorORM.FindAll(&users); err != nil {
		log.Fatal("Failed to find all users:", err)
	}
	fmt.Printf("Total users: %d\n", len(users))

	// Query with conditions
	var publishedPosts []Post
	if err := gorORM.Query(&Post{}).Where("published = ?", true).FindAll(&publishedPosts); err != nil {
		log.Fatal("Failed to find published posts:", err)
	}
	fmt.Printf("Published posts: %d\n", len(publishedPosts))

	// Count posts
	count, err := gorORM.Query(&Post{}).Where("user_id = ?", user.ID).Count()
	if err != nil {
		log.Fatal("Failed to count posts:", err)
	}
	fmt.Printf("Total posts by user: %d\n", count)

	// Update user
	foundUser.FirstName = "Jonathan"
	if err := gorORM.Update(&foundUser); err != nil {
		log.Fatal("Failed to update user:", err)
	}
	fmt.Printf("✅ Updated user name to: %s\n", foundUser.FirstName)

	// Transaction example
	fmt.Println("\n💼 Transaction Example:")
	err = gorORM.Transaction(ctx, func(tx gor.Transaction) error {
		// Create multiple posts in a transaction
		for i := 0; i < 3; i++ {
			post := &Post{
				Title:     fmt.Sprintf("Transaction Post %d", i+1),
				Content:   fmt.Sprintf("Content for post %d", i+1),
				UserID:    user.ID,
				Published: true,
			}
			if err := tx.Create(post); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		log.Fatal("Transaction failed:", err)
	}
	fmt.Println("✅ Transaction completed successfully")

	// Final count
	finalCount, err := gorORM.Query(&Post{}).Count()
	if err != nil {
		log.Fatal("Failed to get final count:", err)
	}
	fmt.Printf("📈 Final post count: %d\n", finalCount)

	// Aggregation examples
	fmt.Println("\n📊 Aggregation Examples:")

	// Get latest post
	var latestPost Post
	if err := gorORM.Query(&Post{}).OrderDesc("created_at").First(&latestPost); err != nil {
		log.Fatal("Failed to get latest post:", err)
	}
	fmt.Printf("Latest post: %s\n", latestPost.Title)

	// Bulk operations
	fmt.Println("\n🔄 Bulk Operations:")

	// Bulk update
	affected, err := gorORM.Query(&Post{}).Where("published = ?", false).UpdateAll(map[string]interface{}{
		"published":  true,
		"updated_at": time.Now(),
	})
	if err != nil {
		log.Fatal("Failed to bulk update:", err)
	}
	fmt.Printf("✅ Bulk updated %d posts\n", affected)

	fmt.Println("\n🎉 ORM example completed successfully!")
}