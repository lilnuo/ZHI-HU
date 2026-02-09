package init

import (
	"fmt"
	"go-zhihu/internal/handler"
	"go-zhihu/internal/middleware"
	"log"

	"github.com/gin-gonic/gin"
)

func SetRoute(r *gin.Engine, httpHandler *handler.Handler) {

	publicGroup := r.Group("/api/v1")
	{
		publicGroup.POST("/register", httpHandler.Register)
		publicGroup.POST("/login", httpHandler.Login)
		publicGroup.GET("/posts/search", httpHandler.Search)
		publicGroup.GET("/posts/ranking", httpHandler.GetLeaderboard)
	}
	authGroup := r.Group("/user")
	authGroup.Use(middleware.AuthMiddleware())
	{
		writerGroup := authGroup.Group("/")
		//user
		writerGroup.PUT("profile", httpHandler.UpdateProfile)
		//chapter
		writerGroup.GET("/posts/drafts", httpHandler.GetDrafts)
		writerGroup.GET("/posts/posts_lists", httpHandler.GetLatestPosts)
		writerGroup.POST("/posts", httpHandler.CreatPost)
		authGroup.GET("posts/:id", httpHandler.GetPostDetail) //只读操作，不限流
		writerGroup.PUT("/posts/:id", httpHandler.UpdatePost)
		writerGroup.DELETE("/posts/:id", httpHandler.DeletePost)
		//people interaction
		writerGroup.POST("/follow/:id", httpHandler.FollowUser)
		writerGroup.POST("/unfollow/:id", httpHandler.UnFollowUser)
		writerGroup.POST("/like/:post_id", httpHandler.ToggleLike)
		//comment
		writerGroup.GET("/posts/comments", httpHandler.GetComments)
		writerGroup.POST("/posts/:post_id/comments", httpHandler.AddComment)
		//feed
		authGroup.GET("/feed", httpHandler.GetFeed)
		//administer
		writerGroup.POST("/ban/:user:id", httpHandler.BanUser)
	}
	fmt.Println("start service on 8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start service:", err)
	}
}
