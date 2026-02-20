package service

import (
	"context"
	"go-zhihu/internal/model"
	"go-zhihu/internal/repository"

	"gorm.io/gorm"
)

type NotificationService struct {
	repo *repository.NotificationRepository
}

func NewNotificationService(repo *repository.NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

// 信息通知
func (s *NotificationService) sendNotification(ctx context.Context, tx *gorm.DB, recipientID, actorID uint, nType int, content string, targetID uint) {
	if recipientID == actorID {
		return
	}
	notification := &model.Notification{
		RecipientID: recipientID,
		ActorID:     actorID,
		Type:        nType,
		Content:     content,
		TargetID:    targetID,
		IsRead:      false,
	}
	_ = s.repo.CreateNotification(ctx, tx, notification)
}

// 系统通知
func (s *NotificationService) SendSystemNotice(ctx context.Context, tx *gorm.DB, recipientID uint, content string) error {
	notification := &model.Notification{
		RecipientID: recipientID,
		ActorID:     0,
		Type:        model.NotifyTypeSystem,
		Content:     content,
		TargetID:    0,
		IsRead:      false,
	}
	return s.repo.CreateNotification(ctx, tx, notification)
}
func (s *NotificationService) GetNotifications(ctx context.Context, tx *gorm.DB, userID uint, page, pageSize int) ([]model.Notification, error) {
	offset := (page - 1) * pageSize
	return s.repo.GetNotifications(ctx, tx, userID, offset, pageSize)
}
func (s *NotificationService) GetUnreadCount(ctx context.Context, tx *gorm.DB, userID uint) (int64, error) {
	return s.repo.GetUnreadCount(ctx, tx, userID)
}
func (s *NotificationService) MarkNotificationRead(ctx context.Context, tx *gorm.DB, notificationID, userID uint) error {
	return s.repo.MarkAsRead(ctx, tx, notificationID, userID)
}
func (s *NotificationService) MarkAllNotificationsRead(ctx context.Context, tx *gorm.DB, userID uint) error {
	return s.repo.MarkAllAsRead(ctx, tx, userID)
}
