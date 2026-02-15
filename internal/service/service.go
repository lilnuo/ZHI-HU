package service

import (
	"go-zhihu/internal/repository"
	"math/rand"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type Service struct {
	User         *UserService
	Post         *PostService
	Interaction  *InteractionService
	Relation     *RelationService
	Feed         *FeedService
	Message      *MessageService
	Notification *NotificationService
}

func NewService(db *gorm.DB, rdb *redis.Client, jwtSecret string) *Service {
	userRepo := repository.NewUserRepository(db)
	postRepo := repository.NewPostRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	relationRepo := repository.NewRelationRepository(db)
	likeRepo := repository.NewLikeRepository(db)
	feedRepo := repository.NewFeedRepository(db)
	connRepo := repository.NewConnectionRepository(db)
	notifyRepo := repository.NewNotificationRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	notifySvc := NewNotificationService(notifyRepo)
	feedSvc := NewFeedService(feedRepo, postRepo, relationRepo, rdb)
	return &Service{
		User:        NewUserService(userRepo, notifySvc, jwtSecret),
		Post:        NewPostService(postRepo, likeRepo, feedSvc, rdb),
		Interaction: NewInteractionService(likeRepo, commentRepo, postRepo, connRepo, notifySvc, db),
		Relation:    NewRelationService(relationRepo, userRepo, feedSvc, notifySvc),
		Feed:        feedSvc,
		Message:     NewMessageService(messageRepo, notifySvc),
	}
}

const (
	FeedKeyPrefix = "feed:user:"
	FeedPushLimit = 100
)

// 用随机过期方式来防止缓存雪崩
func getRandomExpire(base time.Duration) time.Duration {
	return base + time.Duration(rand.Intn(300))*time.Second
}
