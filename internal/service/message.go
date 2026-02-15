package service

import (
	"fmt"
	"go-zhihu/internal/model"
	"go-zhihu/internal/repository"
	"go-zhihu/pkg/e"
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
func (s *MessageService) GetChatHistory(userID, peerID uint, page, pageSize int) ([]model.Message, error) {
	sessionID := generateSessionID(userID, peerID)
	offset := (page - 1) * pageSize
	messages, err := s.repo.GetMessageBySession(sessionID, offset, pageSize)
	if err != nil {
		return nil, e.ErrServer
	}
	_ = s.repo.MarkMessagesAsRead(sessionID, userID)
	return messages, nil
}

// 获取会话列表
func (s *MessageService) GetConversations(userID uint) ([]model.Message, error) {
	return s.repo.GetConversations(userID)
}

// total Unread
func (s *MessageService) GetTotalUnread(userID uint) (map[string]int64, error) {
	notifyCount, err := s.notify.repo.GetUnreadCount(userID)
	if err != nil {
		notifyCount = 0
	}
	msgCount, err := s.repo.GetUnreadCountByUser(userID)
	if err != nil {
		msgCount = 0
	}
	return map[string]int64{
		"notification_unread": notifyCount,
		"message_unread":      msgCount,
		"total_unread":        notifyCount + msgCount,
	}, nil
}
