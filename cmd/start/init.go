package start

import (
	"fmt"
	"go-zhihu/internal/handler"
	"go-zhihu/internal/middleware"
	"go-zhihu/internal/repository"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetRoute(r *gin.Engine, httpHandler *handler.Handler, repos *repository.Repositories, db *gorm.DB) {
	// CORS 配置 - 允许前端跨域请求
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	publicGroup := r.Group("")
	{
		publicGroup.POST("/register", httpHandler.Register)
		publicGroup.POST("/login", httpHandler.Login)
		publicGroup.GET("/posts/search", httpHandler.Search)
		publicGroup.GET("/posts/ranking", httpHandler.GetLeaderboard)
	}
	authGroup := r.Group("/user")
	authGroup.Use(middleware.AuthMiddleware())
	authGroup.Use(middleware.CheckStatus(repos.User, db))
	{
		writerGroup := authGroup.Group("/")
		//user社交关系
		//people interaction
		writerGroup.POST("follow/:id", httpHandler.FollowUser)
		writerGroup.POST("unfollow/:id", httpHandler.UnFollowUser)
		writerGroup.GET("followers", httpHandler.GetFollowers)
		writerGroup.GET("following", httpHandler.GetFollowees)
		//用户信息
		writerGroup.PUT("profile", httpHandler.UpdateProfile)
		writerGroup.GET("profile", httpHandler.GetUserProfile)
		writerGroup.GET(":id/posts", httpHandler.GetUserPosts)
		//文章操作
		writerGroup.GET("posts/drafts", httpHandler.GetDrafts)
		writerGroup.GET("posts/lists", httpHandler.GetLatestPosts)
		writerGroup.POST("posts", httpHandler.CreatPost)
		writerGroup.POST("posts/:id/publish", httpHandler.PublishPost)
		writerGroup.PUT("posts/:id", httpHandler.UpdatePost)
		writerGroup.DELETE("posts/:id", httpHandler.DeletePost)
		//文章关注
		writerGroup.POST("connection/:id", httpHandler.ToggleConn)
		writerGroup.POST("connections", httpHandler.GetConn)
		//点赞文章
		writerGroup.POST("like", httpHandler.ToggleLike)
		//comment
		writerGroup.GET("posts/:post_id", httpHandler.GetComments)
		writerGroup.POST("posts/:id/comments", httpHandler.AddComment)
		//通知中心
		writerGroup.GET("notifications", httpHandler.GetNotifications)
		writerGroup.GET("notifications/unread", httpHandler.GetUnreadCount)
		writerGroup.PUT("notifications/read/:id", httpHandler.MarkNotificationRead)
		writerGroup.PUT("notifications/read/read_all", httpHandler.MarkAllRead)
		//私信
		writerGroup.POST("messages", httpHandler.SendMsg)
		writerGroup.GET("messages/conversations", httpHandler.GetConversations)
		writerGroup.GET("messages/unread", httpHandler.GetTotalUnread)
		writerGroup.GET("messages/:id", httpHandler.GetChatHistory)
	}
	usersGroup := r.Group("/users")
	{
		usersGroup.GET("/:id/profile", httpHandler.GetUserProfile)
		usersGroup.GET("/:id/posts", httpHandler.GetUserPosts)
	}
	publicGroup.GET("/posts/:id", httpHandler.GetPostDetail)
	authGroup.GET("feed", httpHandler.GetFeed)

	//administer
	adminGroup := authGroup.Group("/admin")
	adminGroup.Use(middleware.AdminMiddleware())
	{
		adminGroup.POST("/ban/:id", httpHandler.BanUser)
		adminGroup.POST("/unban/:id", httpHandler.UnbanUser)
	}
	fmt.Println("start service on 8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start service:", err)
	}
}
