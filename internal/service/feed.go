package service

import (
	"context"
	"fmt"
	"go-zhihu/internal/model"
	"go-zhihu/internal/repository"
	"go-zhihu/pkg/e"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type FeedService struct {
	feedRepo     *repository.FeedRepository
	postRepo     *repository.PostRepository
	relationRepo *repository.RelationRepository
	rdb          *redis.Client
}

func NewFeedService(feed *repository.FeedRepository, post *repository.PostRepository, relation *repository.RelationRepository, rdb *redis.Client) *FeedService {
	return &FeedService{feedRepo: feed, postRepo: post, relationRepo: relation, rdb: rdb}
}

// 异步将被关注者的文章推送到关注者的时间线
func (s *FeedService) PushPostsToFeed(ct context.Context, tx *gorm.DB, followerID, followeeID uint) {
	posts, err := s.postRepo.FindRecentPostIDsByAuthor(ct, tx, followeeID, FeedPushLimit)
	if err != nil {
		return
	}
	if len(posts) == 0 {
		return
	}
	ctx := context.Background()
	key := fmt.Sprintf("%s%d", FeedKeyPrefix, followerID)

	pipe := s.rdb.Pipeline()
	for _, post := range posts {
		pipe.ZAdd(ctx, key, &redis.Z{
			Score:  float64(post.CreatedAt.Unix()),
			Member: post.ID,
		})
	}
	pipe.Expire(ctx, key, time.Hour*24*7)
	_, _ = pipe.Exec(ctx)
}

// 推送feed
func (s *PostService) DistributePostToFollowers(ct context.Context, tx *gorm.DB, post *model.Post) {
	ctx := context.Background()
	followerIDs, err := s.relation.GetFollowerIDs(ct, tx, post.AuthorID)
	if err != nil {
		return
	}
	pipe := s.rdb.Pipeline()
	for _, fid := range followerIDs {
		key := fmt.Sprintf("%s%d", FeedKeyPrefix, fid)
		pipe.ZAdd(ctx, key, &redis.Z{
			Score:  float64(post.CreatedAt.Unix()),
			Member: post.ID,
		}) //可以限制用户关注人数
	}
	_, _ = pipe.Exec(ctx)
}
func (s *FeedService) GetFeed(ct context.Context, tx *gorm.DB, userID uint, page, pageSize int) ([]model.Post, error) {
	ctx := context.Background()
	key := fmt.Sprintf("%s%d", FeedKeyPrefix, userID)
	start := int64((page - 1) * pageSize)
	end := start + int64(pageSize) - 1
	postIDs, err := s.rdb.ZRevRange(ctx, key, start, end).Result()
	if err == nil && len(postIDs) > 0 {
		posts, err := s.postRepo.FindPostsByIDs(ct, tx, postIDs)
		if err != nil {
			return nil, err
		}
		postMap := make(map[uint]model.Post)
		for _, p := range posts {
			postMap[p.ID] = p
		}
		sortedPosts := make([]model.Post, 0, len(postIDs))
		for _, idStr := range postIDs {
			id, _ := strconv.ParseUint(idStr, 10, 64)
			if p, ok := postMap[uint(id)]; ok {
				sortedPosts = append(sortedPosts, p)
			}
		}
		return sortedPosts, nil
	}
	followeeIDs, err := s.relationRepo.GetFolloweeIDs(ct, tx, userID)
	if err != nil {
		return nil, e.ErrServer
	}
	if len(followeeIDs) == 0 {
		return []model.Post{}, nil
	}
	offset := (page - 1) * pageSize
	return s.feedRepo.GetFeedByUserIDs(ct, tx, followeeIDs, offset, pageSize)
}
