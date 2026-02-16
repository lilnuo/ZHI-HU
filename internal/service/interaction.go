package service

import (
	"context"
	"errors"
	"go-zhihu/internal/model"
	"go-zhihu/internal/repository"
	"go-zhihu/pkg/e"

	"gorm.io/gorm"
)

type InteractionService struct {
	likeRepo    *repository.LikeRepository
	commentRepo *repository.CommentRepository
	postRepo    *repository.PostRepository
	connRepo    *repository.ConnectRepository
	notify      *NotificationService
	db          *gorm.DB
}

func NewInteractionService(like *repository.LikeRepository, comment *repository.CommentRepository, post *repository.PostRepository, conn *repository.ConnectRepository, notify *NotificationService, db *gorm.DB) *InteractionService {
	return &InteractionService{likeRepo: like, commentRepo: comment, postRepo: post, connRepo: conn, notify: notify, db: db}
}

// 查看评论
func (s *InteractionService) GetComments(ctx context.Context, tx *gorm.DB, postID uint) ([]model.Comment, error) {
	_, err := s.postRepo.FindPostByID(ctx, tx, postID)
	if err != nil {
		return nil, e.ErrPostNotFound
	}
	return s.commentRepo.GetCommentByPostID(ctx, tx, postID)
}

func (s *InteractionService) ToggleLike(ctx context.Context, tx *gorm.DB, userID uint, targetID uint, targetType int) error {
	if targetType != model.TargetTypePost && targetType != model.TargetTypeComment {
		return e.ErrInvalidArgs
	}
	var scoreDelta float64
	var isNewAction bool
	var authorID uint
	const (
		likePostScore    = 10.0
		likeCommentScore = 5.0
	) //开启事务
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existingLike, err := s.likeRepo.FindLike(ctx, userID, targetID, targetType, tx)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if existingLike != nil {
			isNewAction = false
			if err := s.likeRepo.RemoveLike(ctx, tx, userID, targetID, targetType); err != nil {
				return err
			}
			if targetType == model.TargetTypePost {
				scoreDelta = -likePostScore
				if err := s.postRepo.UpdateHotScore(ctx, tx, targetID, scoreDelta); err != nil {
					return err
				}
				comment, err := s.commentRepo.FindCommentByID(ctx, tx, targetID)
				if err != nil {
					return err
				}
				scoreDelta = -likeCommentScore
				authorID = comment.PostID
				if err := s.postRepo.UpdateHotScore(ctx, tx, authorID, scoreDelta); err != nil {
					return err
				}
			} else {
				//新增点赞
				isNewAction = true
				like := &model.Like{UserID: userID, TargetID: targetID, Type: targetType}
				if err := s.likeRepo.AddLike(like, ctx, tx); err != nil {
					return err
				}
				if targetType == model.TargetTypePost {
					scoreDelta = likePostScore
					post, err := s.postRepo.FindPostByID(ctx, tx, targetID)
					if err != nil {
						return err
					}
					authorID = post.AuthorID
					if err := s.postRepo.UpdateHotScore(ctx, tx, targetID, scoreDelta); err != nil {
						return err
					}
				} else {
					comment, err := s.commentRepo.FindCommentByID(ctx, tx, targetID)
					if err != nil {
						return err
					}
					authorID = comment.AuthorID
					scoreDelta = likeCommentScore
					if err := s.postRepo.UpdateHotScore(ctx, tx, comment.PostID, scoreDelta); err != nil {
						return err
					}
				}
			}
		} //事务提交后操作
		return nil
	})
	if err != nil {
		return e.ErrServer
	}
	//异步发送通知
	if isNewAction {
		content := "赞了你的文章"
		if targetType == model.TargetTypeComment {
			content = "赞了你的评论"
		}
		s.notify.sendNotification(ctx, tx, authorID, userID, model.NotifyTypeLike, content, targetID)
	}
	return nil
}

func (s *InteractionService) AddComment(ctx context.Context, tx *gorm.DB, postID, authorID uint, content string) error {
	comment := &model.Comment{
		PostID:   postID,
		AuthorID: authorID,
		Content:  content,
	}
	err := s.commentRepo.CreateComment(ctx, tx, comment)
	if err != nil {
		return e.ErrServer
	}
	post, _ := s.postRepo.FindPostByID(ctx, tx, postID)
	if post != nil {
		newScore := post.Hotscore + 5
		_ = s.postRepo.UpdateHotScore(ctx, tx, postID, newScore)
		s.notify.sendNotification(ctx, tx, post.AuthorID, authorID, model.NotifyTypeComment, "评论了你的文章", postID)
	}
	return nil
}

// 添加收藏关注列表
func (s *InteractionService) ToggleConn(ctx context.Context, tx *gorm.DB, userID, postID uint) error {
	_, err := s.postRepo.FindPostByID(ctx, tx, postID)
	if err != nil {
		return e.ErrPostNotFound
	}
	isConn, err := s.connRepo.IsConn(ctx, tx, userID, postID)
	if err != nil {
		return e.ErrServer
	}
	if isConn {
		return s.connRepo.RemoveConn(ctx, tx, userID, postID)
	}
	return s.connRepo.AddConnection(ctx, tx, userID, postID)
}

// 获取收藏列表
func (s *InteractionService) GetConn(ctx context.Context, tx *gorm.DB, userID uint, page, pageSze int) ([]model.Post, error) {
	offset := (page - 1) * pageSze
	return s.connRepo.GetConnByUser(ctx, tx, userID, offset, pageSze)
}
