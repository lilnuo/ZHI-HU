package service

import (
	"context"
	"fmt"
	"go-zhihu/internal/model"
	"go-zhihu/internal/repository"
	"go-zhihu/pkg/e"

	"gorm.io/gorm"
)

type MessageService struct {
	repo   *repository.MessageRepository
	notify *NotificationService
}

func NewMessageService(repo *repository.MessageRepository, notify *NotificationService) *MessageService {
	return &MessageService{repo: repo, notify: notify}
}

// 私信通知（但没有系统通知）
func generateSessionID(uid1, uid2 uint) string {
	if uid1 < uid2 {
		return fmt.Sprintf("%d_%d", uid1, uid2)
	}
	return fmt.Sprintf("%d_%d", uid2, uid1)
}

// 获取聊天记录
func (s *MessageService) GetChatHistory(ctx context.Context, tx *gorm.DB, userID, peerID uint, page, pageSize int) ([]model.Message, error) {
	sessionID := generateSessionID(userID, peerID)
	offset := (page - 1) * pageSize
	messages, err := s.repo.GetMessageBySession(ctx, tx, sessionID, offset, pageSize)
	if err != nil {
		return nil, e.ErrServer
	}
	_ = s.repo.MarkMessagesAsRead(ctx, tx, sessionID, userID)
	return messages, nil
}

// 获取会话列表
func (s *MessageService) GetConversations(ctx context.Context, tx *gorm.DB, userID uint) ([]model.Message, error) {
	return s.repo.GetConversations(ctx, tx, userID)
}

// total Unread
func (s *MessageService) GetTotalUnread(ctx context.Context, tx *gorm.DB, userID uint) (map[string]int64, error) {
	notifyCount, err := s.notify.repo.GetUnreadCount(ctx, tx, userID)
	if err != nil {
		notifyCount = 0
	}
	msgCount, err := s.repo.GetUnreadCountByUser(ctx, tx, userID)
	if err != nil {
		msgCount = 0
	}
	return map[string]int64{
		"notification_unread": notifyCount,
		"message_unread":      msgCount,
		"total_unread":        notifyCount + msgCount,
	}, nil
}
