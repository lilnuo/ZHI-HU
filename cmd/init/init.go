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
		writerGroup.GET("followers", httpHandler.GetFollowers)
		writerGroup.GET("following", httpHandler.GetFollowees)
		writerGroup.PUT("profile", httpHandler.UpdateProfile)
		//chapter
		writerGroup.GET("/posts/drafts", httpHandler.GetDrafts)
		writerGroup.GET("/posts/posts_lists", httpHandler.GetLatestPosts)
		writerGroup.POST("/posts", httpHandler.CreatPost)

		authGroup.GET("posts/:id", httpHandler.GetPostDetail) //只读操作，不限流
		writerGroup.POST("/posts/:id/publish", httpHandler.PublishPost)
		writerGroup.PUT("/posts/:id", httpHandler.UpdatePost)
		writerGroup.DELETE("/posts/:id", httpHandler.DeletePost)
		//people interaction
		writerGroup.POST("/follow/:id", httpHandler.FollowUser)
		writerGroup.POST("/unfollow/:id", httpHandler.UnFollowUser)
		//文章关注
		writerGroup.POST("/connection/:id", httpHandler.ToggleConn)
		writerGroup.POST("/connections", httpHandler.GetConn)
		//点赞文章
		writerGroup.POST("/like", httpHandler.ToggleLike)
		//comment
		writerGroup.GET("/posts/comments", httpHandler.GetComments)
		writerGroup.POST("/posts/:post_id/comments", httpHandler.AddComment)
		//feed
		authGroup.GET("/feed", httpHandler.GetFeed)
		//通知中心
		writerGroup.GET("/notifications", httpHandler.GetNotifications)
		writerGroup.GET("/notifications/unread", httpHandler.GetUnreadCount)
		writerGroup.PUT("/notifications/read/:id", httpHandler.MarkNotificationRead)
		writerGroup.PUT("/notifications/read/read_all", httpHandler.MarkAllRead)
		//私信
		writerGroup.POST("/messages", httpHandler.SendMsg)
		writerGroup.GET("/messages/conversations", httpHandler.GetConversations)
		writerGroup.GET("/messages/unread", httpHandler.GetTotalUnread)
		writerGroup.GET("messages/:id", httpHandler.GetChatHistory)
		//administer
		adminGroup := authGroup.Group("/")
		adminGroup.Use(middleware.AdminMiddleware())
		{
			adminGroup.POST("/ban/:id", httpHandler.BanUser)
			adminGroup.POST("/unban/:id", httpHandler.UnbanUser)
		}
	}
	fmt.Println("start service on 8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start service:", err)
	}
}
