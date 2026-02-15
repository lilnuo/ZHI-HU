package service

import (
	"go-zhihu/internal/model"
	"go-zhihu/internal/repository"
	"go-zhihu/pkg/e"
)

type NotificationService struct {
	repo *repository.NotificationRepository
}

func NewNotificationService(repo *repository.NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

// 信息通知
func (s *NotificationService) sendNotification(recipientID, actorID uint, nType int, content string, targetID uint) {
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
	_ = s.repo.CreateNotification(notification)
}
func (s *MessageService) SendMessage(senderID, receiverID uint, content string) error {
	if senderID == receiverID {
		return e.ErrSelfAction
	}
	//可以写好友状态，看是否可以发私信
	sessionID := generateSessionID(senderID, receiverID)
	msg := &model.Message{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Content:    content,
		Session:    sessionID,
		IsRead:     false,
	}
	if err := s.repo.CreateMessage(msg); err != nil {
		return e.ErrServer
	}
	s.notify.sendNotification(receiverID, senderID, model.NotifyTypeMessage, "给你发来一条私信", 0)
	return nil
}

// 系统通知
func (s *NotificationService) SendSystemNotice(recipientID uint, content string) error {
	notification := &model.Notification{
		RecipientID: recipientID,
		ActorID:     0,
		Type:        model.NotifyTypeSystem,
		Content:     content,
		TargetID:    0,
		IsRead:      false,
	}
	return s.repo.CreateNotification(notification)
}
func (s *NotificationService) GetNotifications(userID uint, page, pageSize int) ([]model.Notification, error) {
	offset := (page - 1) * pageSize
	return s.repo.GetNotifications(userID, offset, pageSize)
}
func (s *NotificationService) GetUnreadCount(userID uint) (int64, error) {
	return s.repo.GetUnreadCount(userID)
}
func (s *NotificationService) MarkNotificationRead(notificationID, userID uint) error {
	return s.repo.MarkAsRead(notificationID, userID)
}
func (s *NotificationService) MarkAllNotificationsRead(userID uint) error {
	return s.repo.MarkAllAsRead(userID)
}
