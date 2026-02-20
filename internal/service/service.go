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

func NewService(db *gorm.DB, rdb *redis.Client, repos *repository.Repositories, jwtSecret string) *Service {

	notifySvc := NewNotificationService(repos.Notification)
	feedSvc := NewFeedService(repos.Feed, repos.Post, repos.Relation, rdb)
	return &Service{
		User:        NewUserService(repos.User, notifySvc, rdb, jwtSecret),
		Post:        NewPostService(repos.Post, repos.Like, feedSvc, rdb),
		Interaction: NewInteractionService(repos.Like, repos.Comment, repos.Post, repos.Connection, notifySvc, db),
		Relation:    NewRelationService(repos.Relation, repos.User, feedSvc, notifySvc),
		Feed:        feedSvc,
		Message:     NewMessageService(repos.Message, notifySvc),
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
