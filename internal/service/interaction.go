package service

import (
	"go-zhihu/internal/model"
	"go-zhihu/internal/repository"
	"go-zhihu/pkg/e"
)

type InteractionService struct {
	likeRepo    *repository.LikeRepository
	commentRepo *repository.CommentRepository
	postRepo    *repository.PostRepository
	connRepo    *repository.ConnectRepository
	notify      *NotificationService
}

func NewInteractionService(like *repository.LikeRepository, comment *repository.CommentRepository, post *repository.PostRepository, conn *repository.ConnectRepository, notify *NotificationService) *InteractionService {
	return &InteractionService{likeRepo: like, commentRepo: comment, postRepo: post, connRepo: conn, notify: notify}
}

// 查看评论
func (s *InteractionService) GetComments(postID uint) ([]model.Comment, error) {
	_, err := s.postRepo.FindPostByID(postID)
	if err != nil {
		return nil, e.ErrPostNotFound
	}
	return s.commentRepo.GetCommentByPostID(postID)
}

func (s *InteractionService) ToggleLike(userID uint, targetID uint, targetType int) error {
	if targetType == 1 {
		post, err := s.postRepo.FindPostByID(targetID)
		if err != nil {
			return e.ErrPostNotFound
		}
		_, isNewAction, err := s.likeRepo.ToggleLikeWithTx(userID, targetID, targetType)
		if err != nil {
			return e.ErrServer
		}
		if isNewAction {
			s.notify.sendNotification(post.AuthorID, userID, model.NotifyTypeLike, "赞了你的文章", targetID)
		}
	} else if targetID == 2 {
		comment, err := s.commentRepo.FindCommentByID(targetID)
		if err != nil {
			return e.ErrInvalidArgs //error comment not found
		}
		_, isNewAction, err := s.likeRepo.ToggleLikeWithTx(userID, targetID, targetType)
		if err != nil {
			return e.ErrServer
		}
		if isNewAction {
			s.notify.sendNotification(comment.AuthorID, userID, model.NotifyTypeLike, "赞了你的评论", comment.ID)
		}
	} else {
		return e.ErrInvalidArgs
	}
	return nil
}

func (s *InteractionService) AddComment(postID, authorID uint, content string) error {
	comment := &model.Comment{
		PostID:   postID,
		AuthorID: authorID,
		Content:  content,
	}
	err := s.commentRepo.CreateComment(comment)
	if err != nil {
		return e.ErrServer
	}
	post, _ := s.postRepo.FindPostByID(postID)
	if post != nil {
		newScore := post.Hotscore + 5
		_ = s.postRepo.UpdateHotScore(postID, newScore)
		s.notify.sendNotification(post.AuthorID, authorID, model.NotifyTypeComment, "评论了你的文章", postID)
	}
	return nil
}

// 添加收藏关注列表
func (s *InteractionService) ToggleConn(userID, postID uint) error {
	_, err := s.postRepo.FindPostByID(postID)
	if err != nil {
		return e.ErrPostNotFound
	}
	isConn, err := s.connRepo.IsConn(userID, postID)
	if err != nil {
		return e.ErrServer
	}
	if isConn {
		return s.connRepo.RemoveConn(userID, postID)
	}
	return s.connRepo.AddConnection(userID, postID)
}

// 获取收藏列表
func (s *InteractionService) GetConn(userID uint, page, pageSze int) ([]model.Post, error) {
	offset := (page - 1) * pageSze
	return s.connRepo.GetConnByUser(userID, offset, pageSze)
}
