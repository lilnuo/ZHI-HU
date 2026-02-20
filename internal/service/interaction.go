package service

import (
	"context"
	"errors"
	"go-zhihu/internal/model"
	"go-zhihu/internal/repository"
	"go-zhihu/pkg/e"
	"log"

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
	// 参数校验
	if targetType != model.TargetTypePost && targetType != model.TargetTypeComment {
		return e.ErrInvalidArgs
	}

	const (
		likePostScore    = 10.0
		likeCommentScore = 5.0
	)

	var (
		scoreDelta  float64
		isNewAction bool
		authorID    uint
		postID      uint // 用于评论点赞时更新文章热度
	)

	// 使用事务
	err := s.db.WithContext(ctx).Transaction(func(txFn *gorm.DB) error {
		// 查找是否已存在点赞
		existingLike, err := s.likeRepo.FindLike(ctx, userID, targetID, targetType, txFn)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if existingLike != nil {
			isNewAction = false

			// 删除点赞记录
			if err := s.likeRepo.RemoveLike(ctx, txFn, userID, targetID, targetType); err != nil {
				return err
			}

			if targetType == model.TargetTypePost {
				// 取消文章点赞
				scoreDelta = -likePostScore
				if err := s.postRepo.UpdateHotScore(ctx, txFn, targetID, scoreDelta); err != nil {
					return err
				}
				// 获取作者ID用于通知（可选，取消点赞通常不发通知）
				post, err := s.postRepo.FindPostByID(ctx, txFn, targetID)
				if err == nil {
					authorID = post.AuthorID
				}
			} else if targetType == model.TargetTypeComment {
				// 取消评论点赞
				comment, err := s.commentRepo.FindCommentByID(ctx, txFn, targetID)
				if err != nil {
					return err
				}
				scoreDelta = -likeCommentScore
				postID = comment.PostID
				authorID = comment.AuthorID
				if err := s.postRepo.UpdateHotScore(ctx, txFn, postID, scoreDelta); err != nil {
					return err
				}
			}
		} else {
			// ========== 新增点赞逻辑 ==========
			isNewAction = true

			// 创建点赞记录
			like := &model.Like{
				UserID:   userID,
				TargetID: targetID,
				Type:     targetType,
			}
			if err := s.likeRepo.AddLike(like, ctx, txFn); err != nil {
				return err
			}

			if targetType == model.TargetTypePost {
				// 文章点赞
				post, err := s.postRepo.FindPostByID(ctx, txFn, targetID)
				if err != nil {
					return err
				}
				authorID = post.AuthorID
				scoreDelta = likePostScore
				if err := s.postRepo.UpdateHotScore(ctx, txFn, targetID, scoreDelta); err != nil {
					return err
				}
			} else if targetType == model.TargetTypeComment {
				// 评论点赞
				comment, err := s.commentRepo.FindCommentByID(ctx, txFn, targetID)
				if err != nil {
					return err
				}
				authorID = comment.AuthorID
				postID = comment.PostID
				scoreDelta = likeCommentScore
				if err := s.postRepo.UpdateHotScore(ctx, txFn, postID, scoreDelta); err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		return e.ErrServer
	}

	// 异步发送通知（只在新点赞时发送）
	if isNewAction && authorID != userID {
		content := "赞了你的文章"
		if targetType == model.TargetTypeComment {
			content = "赞了你的评论"
		}
		// 使用新的 context 避免原 context 超时
		go func() {
			bgCtx := context.Background()
			s.notify.sendNotification(bgCtx, tx, authorID, userID, model.NotifyTypeLike, content, targetID)
		}()
	}

	return nil
}

func (s *InteractionService) AddComment(ctx context.Context, tx *gorm.DB, postID, authorID uint, content string) error {
	if content == "" {
		return e.ErrInvalidArgs
	}
	post, err := s.postRepo.FindPostByID(ctx, tx, postID)
	if err != nil {
		return e.ErrPostNotFound
	}
	comment := &model.Comment{
		PostID:   postID,
		AuthorID: authorID,
		Content:  content,
	}
	err = s.commentRepo.CreateComment(ctx, tx, comment)
	if err != nil {
		return e.ErrServer
	}
	const commentScoreDelta = 5.0
	if err := s.postRepo.UpdateHotScore(ctx, tx, postID, commentScoreDelta); err != nil {
		// 热度更新失败不影响评论创建，记录日志即可
		log.Printf("failed to update hot score: %v", err)
	}

	// 发送通知（不要通知自己）
	if post.AuthorID != authorID {
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
