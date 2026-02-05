package main

import (
	"fmt"
	"go-zhihu/internal/handler"
	"go-zhihu/internal/middleware"
	"go-zhihu/internal/model"
	"go-zhihu/internal/repository"
	"go-zhihu/internal/service"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	dsn = "root:nssg0822@tcp(127.0.0.1:3306)/zhihu?charset=utf8mb4&parseTime=True&loc=Local"
)

func main() {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect database:", err)
	}
	err = db.AutoMigrate(
		&model.User{},
		&model.Post{},
		&model.Like{},
		&model.Comment{},
		&model.Relation{},
	)
	if err != nil {
		log.Fatal("Failed to connect database:", err)
	}
	repos := repository.NewRepositories(db)
	socialService := service.NewUserService(
		repos.Relation,
		repos.Like,
		repos.Post,
		repos.Feed,
		repos.Comment,
		repos.User)
	httpHandler := handler.NewUserHandler(socialService)
	r := gin.Default()
	publicGroup := r.Group("/api/v1")
	{
		publicGroup.POST("/register", httpHandler.Register)
		publicGroup.POST("/login", httpHandler.Login)
		publicGroup.GET("/posts/search", httpHandler.Search)
	}
	privateGroup := r.Group("/api/v1")
	privateGroup.Use(middleware.AuthMiddleware())
	{ //chapter
		privateGroup.POST("/posts", httpHandler.CreatPost)
		privateGroup.GET("posts/:id", httpHandler.GetPostDetail)
		privateGroup.PUT("/posts/:id", httpHandler.UpdatePost)
		privateGroup.DELETE("/posts/:id", httpHandler.DeletePost)
		//people interaction
		privateGroup.POST("/follow/:id", httpHandler.FollowUser)
		privateGroup.POST("/unfollow/:id", httpHandler.UnFollowUser)
		privateGroup.POST("/like/:post_id", httpHandler.ToggleLike)
		//comment
		privateGroup.POST("/posts/:post_id/comments", httpHandler.AddComment)
		//feed
		privateGroup.GET("/feed", httpHandler.GetFeed)
		//administer
		privateGroup.POST("/ban/:user:id", httpHandler.BanUser)
	}
	fmt.Println("start service on 8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start service:", err)
	}
}
