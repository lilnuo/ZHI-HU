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

func (s *PostService) CreatePost(authorID uint, title, content string, postType int, status int) error {
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
	if err := s.repo.CreatePost(post); err != nil {
		return e.ErrServer
	}
	if status == 1 {
		go s.DistributePostToFollowers(post)
	}
	return nil
}

//补充普通的最新文章列表

func (s *PostService) GetLatestPosts(page, pageSize int) ([]model.Post, error) {
	offset := (page - 1) * pageSize
	return s.repo.ListPosts(offset, pageSize, "created_at")
}
func (s *PostService) GetPostDetail(postID uint) (*PostDetailVO, error) {
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
	result, err, _ := s.sf.Do(fmt.Sprintf("post:%d", postID), func() (interface{}, error) {
		post, err := s.repo.FindPostByID(postID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, e.ErrPostNotFound) {
				s.rdb.Set(ctx, cacheKey, CacheNullPlaceholder, time.Minute)
				return nil, e.ErrPostNotFound
			}
			return nil, err
		}
		count, err := s.likeRepo.CountLikes(postID)
		postDetail := &PostDetailVO{
			Post:      post,
			LikeCount: count,
		}
		data, _ := json.Marshal(postDetail)
		s.rdb.Set(ctx, cacheKey, data, getRandomExpire(30*time.Minute))
		return postDetail, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*PostDetailVO), nil
}

// 获取草稿箱
func (s *PostService) GetDrafts(userID uint, page, pageSize int) ([]model.Post, error) {
	offset := (page - 1) * pageSize
	return s.repo.ListDrafts(userID, offset, pageSize)
}

// 发布草稿箱
func (s *PostService) PublishPost(postID, authorID uint) error {
	post, err := s.repo.FindPostByID(postID)
	if err != nil {
		return e.ErrPostNotFound
	}
	if post.AuthorID != authorID {
		return e.ErrPermission
	}
	if post.Status != 0 {
		return e.ErrInvalidArgs
	}
	return s.repo.UpdateStatus(postID, 1)
}
func (s *PostService) UpdatePost(postID, authorID uint, title, content string, status *int) error {
	post, err := s.repo.FindPostByID(postID)
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
	s.DeletePostCache(postID)
	return nil
}
func (s *PostService) DeletePost(postID, authorID uint) error {
	post, err := s.repo.FindPostByID(postID)
	if err != nil {
		return e.ErrPostNotFound
	}
	if post.AuthorID != authorID {
		return e.ErrPermission
	}
	post.Status = 2
	if err := s.repo.UpdatePost(post); err != nil {
		return e.ErrServer
	}
	s.DeletePostCache(postID)
	return nil
}
func (s *PostService) DeletePostCache(postID uint) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf(CacheKeyPostDetail, postID)
	s.rdb.Del(ctx, cacheKey)
}
func (s *PostService) Search(keyword string, page, pageSize int) ([]model.Post, error) {
	offset := (page - 1) * pageSize
	return s.repo.SearchPosts(keyword, offset, pageSize)
}

// 获取文章列表
func (s *PostService) GetUserPosts(targetID uint, page, pageSize int) ([]model.Post, error) {
	offset := (page - 1) * pageSize
	return s.repo.ListPublicByAuthorID(targetID, offset, pageSize)
}

// 排行榜补充
func (s *PostService) GetLeaderboard(limit int) ([]model.Post, error) {
	return s.repo.GetLeaderboard(limit)
}
