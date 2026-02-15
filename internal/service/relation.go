package service

import (
	"errors"
	"go-zhihu/internal/model"
	"go-zhihu/internal/repository"
	"go-zhihu/pkg/e"
	"log"

	"gorm.io/gorm"
)

type RelationService struct {
	repo     *repository.RelationRepository
	userRepo *repository.UserRepository
	feed     *FeedService
	notify   *NotificationService
}

func NewRelationService(repo *repository.RelationRepository, user *repository.UserRepository, feed *FeedService, notify *NotificationService) *RelationService {
	return &RelationService{repo: repo, userRepo: user, feed: feed, notify: notify}
}
func (s *RelationService) FollowUser(followerID, followeeID uint) error {
	if followerID == followeeID {
		return e.ErrSelfAction
	}
	err := s.repo.Follow(followerID, followeeID)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return e.ErrAlreadyFollowing
		}
		s.notify.sendNotification(followeeID, followerID, model.NotifyTypeFollow, "关注了你", 0)
		return e.ErrServer
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in pushPostsToFeed: %v", r)
			}
		}()
		s.feed.PushPostsToFeed(followerID, followeeID)
	}()
	s.notify.sendNotification(followeeID, followerID, model.NotifyTypeFollow, "关注了你", 0)
	return nil
}
func (s *RelationService) UnfollowUser(followerID, followeeID uint) error {
	if err := s.repo.Unfollow(followerID, followeeID); err != nil {
		return e.ErrServer
	}
	return nil
}

// 获取当前用户的粉丝列表
func (s *RelationService) GetFollowers(userID uint, page, pageSize int) ([]model.User, error) {
	offset := (page - 1) * pageSize
	return s.repo.GetFollowers(userID, offset, pageSize)
}

// 获取当前用户的关注列表
func (s *RelationService) GetFollowees(userID uint, page, pageSize int) ([]model.User, error) {
	offset := (page - 1) * pageSize
	return s.repo.GetFollowees(userID, offset, pageSize)
}
