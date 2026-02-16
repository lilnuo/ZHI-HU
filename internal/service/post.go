package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-zhihu/internal/model"
	"go-zhihu/internal/repository"
	"go-zhihu/pkg/e"
	"log"
	"time"
	"unicode/utf8"

	"github.com/go-redis/redis/v8"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

type PostService struct {
	repo     *repository.PostRepository
	likeRepo *repository.LikeRepository
	feed     *FeedService
	relation *repository.RelationRepository
	rdb      *redis.Client
	sf       singleflight.Group
}

func NewPostService(repo *repository.PostRepository, likeRepo *repository.LikeRepository, feed *FeedService, rdb *redis.Client) *PostService {
	return &PostService{repo: repo, likeRepo: likeRepo, feed: feed, rdb: rdb}
}

const (
	CacheKeyPostDetail   = "post:detail:%d"
	CacheNullPlaceholder = "NULL"
)

// 处理内容的发布、更新、获取和删

func (s *PostService) CreatePost(ctx context.Context, tx *gorm.DB, authorID uint, title, content string, postType int, status int) error {
	if utf8.RuneCountInString(title) == 0 || utf8.RuneCountInString(title) > 255 {
		return e.ErrInvalidArgs
	}
	if content == "" {
		return e.ErrInvalidArgs
	}
	if postType != 1 && postType != 2 {
		return e.ErrInvalidArgs
	}
	if status != 0 && status != 1 {
		status = 1
	}
	post := &model.Post{
		Title:    title,
		Content:  content,
		Type:     postType, //1.chapter,2.question
		AuthorID: authorID,
		Status:   status, //默认发布
		Hotscore: 0,
	}
	if err := s.repo.CreatePost(ctx, tx, post); err != nil {
		return e.ErrServer
	}
	if status == 1 {
		go s.DistributePostToFollowers(ctx, tx, post)
	}
	return nil
}

//补充普通的最新文章列表

func (s *PostService) GetLatestPosts(ctx context.Context, tx *gorm.DB, page, pageSize int) ([]model.Post, error) {
	offset := (page - 1) * pageSize
	return s.repo.ListPosts(ctx, tx, offset, pageSize, "created_at")
}
func (s *PostService) GetPostDetail(ct context.Context, tx *gorm.DB, postID uint) (*PostDetailVO, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf(CacheKeyPostDetail, postID)
	val, err := s.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		if val == CacheNullPlaceholder {
			return nil, e.ErrPostNotFound
		}
		var postDetail PostDetailVO
		if err := json.Unmarshal([]byte(val), &postDetail); err == nil {
			return &postDetail, nil
		}
	}
	if err != nil && !errors.Is(err, redis.Nil) {
		log.Printf("redis error:%v", err)
	}
	post, err := s.repo.FindPostByID(ct, tx, postID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, e.ErrPostNotFound) {
			s.rdb.Set(ctx, cacheKey, CacheNullPlaceholder, time.Minute)
			return nil, e.ErrPostNotFound
		}
		return nil, e.ErrServer
	}
	count, err := s.likeRepo.CountLikes(ctx, tx, postID)
	postDetail := &PostDetailVO{
		Post:      post,
		LikeCount: count,
	}
	data, _ := json.Marshal(postDetail)
	s.rdb.Set(ctx, cacheKey, data, getRandomExpire(30*time.Minute))
	if err != nil {
		return nil, err
	}
	return &PostDetailVO{Post: post}, nil
}

// 获取草稿箱
func (s *PostService) GetDrafts(ctx context.Context, tx *gorm.DB, userID uint, page, pageSize int) ([]model.Post, error) {
	offset := (page - 1) * pageSize
	return s.repo.ListDrafts(ctx, tx, userID, offset, pageSize)
}

// 发布草稿箱
func (s *PostService) PublishPost(ctx context.Context, tx *gorm.DB, postID, authorID uint) error {
	post, err := s.repo.FindPostByID(ctx, tx, postID)
	if err != nil {
		return e.ErrPostNotFound
	}
	if post.AuthorID != authorID {
		return e.ErrPermission
	}
	if post.Status != 0 {
		return e.ErrInvalidArgs
	}
	return s.repo.UpdateStatus(ctx, tx, postID, 1)
}
func (s *PostService) UpdatePost(ctx context.Context, tx *gorm.DB, postID, authorID uint, title, content string, status *int) error {
	post, err := s.repo.FindPostByID(ctx, tx, postID)
	if err != nil {
		return e.ErrPostNotFound
	}
	if post.AuthorID != authorID {
		return e.ErrPermission
	}
	post.Title = title
	post.Content = content
	if status != nil {
		if *status == 0 || *status == 1 {
			post.Status = *status
		}
	}
	s.DeletePostCache(ctx, tx, postID)
	return nil
}
func (s *PostService) DeletePost(ctx context.Context, tx *gorm.DB, postID, authorID uint) error {
	post, err := s.repo.FindPostByID(ctx, tx, postID)
	if err != nil {
		return e.ErrPostNotFound
	}
	if post.AuthorID != authorID {
		return e.ErrPermission
	}
	post.Status = 2
	if err := s.repo.UpdatePost(ctx, tx, post); err != nil {
		return e.ErrServer
	}
	s.DeletePostCache(ctx, tx, postID)
	return nil
}
func (s *PostService) DeletePostCache(ct context.Context, tx *gorm.DB, postID uint) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf(CacheKeyPostDetail, postID)
	s.rdb.Del(ctx, cacheKey)
}
func (s *PostService) Search(ctx context.Context, tx *gorm.DB, keyword string, page, pageSize int) ([]model.Post, error) {
	offset := (page - 1) * pageSize
	return s.repo.SearchPosts(ctx, tx, keyword, offset, pageSize)
}

// 获取文章列表
func (s *PostService) GetUserPosts(ctx context.Context, tx *gorm.DB, targetID uint, page, pageSize int) ([]model.Post, error) {
	offset := (page - 1) * pageSize
	return s.repo.ListPublicByAuthorID(ctx, tx, targetID, offset, pageSize)
}

// 排行榜补充
func (s *PostService) GetLeaderboard(ctx context.Context, tx *gorm.DB, limit int) ([]model.Post, error) {
	return s.repo.GetLeaderboard(ctx, tx, limit)
}
