package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-zhihu/config"
	"go-zhihu/internal/model"
	"go-zhihu/internal/repository"
	"go-zhihu/pkg/e"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

type SocialService struct {
	RelationRepo     *repository.RelationRepository
	LikeRepo         *repository.LikeRepository
	PostRepo         *repository.PostRepository
	FeedRepo         *repository.FeedRepository
	CommentRepo      *repository.CommentRepository
	UserRepo         *repository.UserRepository
	ConnectionRepo   *repository.ConnectRepository
	NotificationRepo *repository.NotificationRepository
	MessageRepo      *repository.MessageRepository
	rdb              *redis.Client
	sf               singleflight.Group
	secret           string
}

func NewUserService(relationRepo *repository.RelationRepository,
	likeRepo *repository.LikeRepository, postRepo *repository.PostRepository,
	feedRepo *repository.FeedRepository, commentRepo *repository.CommentRepository,
	userRepo *repository.UserRepository, notificationRepo *repository.NotificationRepository,
	messageRepo *repository.MessageRepository,
	rdb *redis.Client, jwtSecret string,
	connectionRepo *repository.ConnectRepository) *SocialService {
	return &SocialService{
		RelationRepo:     relationRepo,
		LikeRepo:         likeRepo,
		PostRepo:         postRepo,
		FeedRepo:         feedRepo,
		CommentRepo:      commentRepo,
		UserRepo:         userRepo,
		ConnectionRepo:   connectionRepo,
		NotificationRepo: notificationRepo,
		MessageRepo:      messageRepo,
		rdb:              rdb,
		secret:           jwtSecret,
	}
}

const (
	CacheKeyPostDetail   = "post:detail:%d"
	CacheNullPlaceholder = "NULL"
)
const (
	FeedKeyPrefix = "feed:user:"
	FeedPushLimit = 100
)

// 用随机过期方式来防止缓存雪崩
func getRandomExpire(base time.Duration) time.Duration {
	return base + time.Duration(rand.Intn(300))*time.Second
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  *model.User `json:"user"`
}

func (s *SocialService) Register(username, password, email string) error {
	_, err := s.UserRepo.FindUsername(username)
	if err == nil {
		return e.ErrorUserExist
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return e.ErrPasswordInstance
	}
	user := &model.User{
		Username: username,
		Password: string(hashedPassword),
		Email:    email,
		Role:     1,
		Status:   1, //默认正常
	}
	if err := s.UserRepo.CreateUser(user); err != nil {
		return e.ErrServer
	}
	return nil
}

// 鉴权加密，环境获取
func (s *SocialService) generateToken(userID uint, username string, role int) (string, error) {
	claims := &jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"role":     strconv.Itoa(role),
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.Setting.JWT.Secret))
}
func (s *SocialService) Login(username, password string) (*LoginResponse, error) {
	user, err := s.UserRepo.FindUsername(username)
	if err != nil {
		return nil, e.ErrUserNotFoundInstance
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, e.ErrPasswordInstance
	}
	if user.Status == 0 {
		return nil, e.ErrUserBanned
	}
	token, err := s.generateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, e.ErrToken
	}
	return &LoginResponse{
		Token: token,
		User:  user,
	}, nil
}

// 个人信息的修改
func (s *SocialService) UpdateProfile(userID uint, avatar, bio string) error {
	updates := make(map[string]interface{})
	if avatar != "" {
		updates["avatar"] = avatar
	}
	if bio != "" {
		updates["bio"] = bio
	}
	if len(updates) == 0 {
		return nil
	}
	if err := s.UserRepo.UpdateProfile(userID, avatar, bio); err != nil {
		return e.ErrServer
	}
	return nil
}

// 处理内容的发布、更新、获取和删

func (s *SocialService) CreatePost(authorID uint, title, content string, postType int, status int) error {
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
	if err := s.PostRepo.CreatePost(post); err != nil {
		return e.ErrServer
	}
	if status == 1 {
		go s.distributePostToFollowers(post)
	}
	return nil
}

//补充普通的最新文章列表

func (s *SocialService) GetLatestPosts(page, pageSize int) ([]model.Post, error) {
	offset := (page - 1) * pageSize
	return s.PostRepo.ListPosts(offset, pageSize, "created_at")
}

// 查看评论
func (s *SocialService) GetComments(postID uint) ([]model.Comment, error) {
	_, err := s.PostRepo.FindPostByID(postID)
	if err != nil {
		return nil, e.ErrPostNotFound
	}
	return s.CommentRepo.GetCommentByPostID(postID)
}
func (s *SocialService) GetPostDetail(postID uint) (*PostDetailVO, error) {
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
		post, err := s.PostRepo.FindPostByID(postID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, e.ErrPostNotFound) {
				s.rdb.Set(ctx, cacheKey, CacheNullPlaceholder, time.Minute)
				return nil, e.ErrPostNotFound
			}
			return nil, err
		}
		count, err := s.LikeRepo.CountLikes(postID)
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
func (s *SocialService) GetDrafts(userID uint, page, pageSize int) ([]model.Post, error) {
	offset := (page - 1) * pageSize
	return s.PostRepo.ListDrafts(userID, offset, pageSize)
}

// 发布草稿箱
func (s *SocialService) PublishPost(postID, authorID uint) error {
	post, err := s.PostRepo.FindPostByID(postID)
	if err != nil {
		return e.ErrPostNotFound
	}
	if post.AuthorID != authorID {
		return e.ErrPermission
	}
	if post.Status != 0 {
		return e.ErrInvalidArgs
	}
	return s.PostRepo.UpdateStatus(postID, 1)
}
func (s *SocialService) UpdatePost(postID, authorID uint, title, content string, status *int) error {
	post, err := s.PostRepo.FindPostByID(postID)
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
	if err := s.PostRepo.UpdatePost(post); err != nil {
		return e.ErrServer
	}
	s.DeletePostCache(postID)
	return nil
}
func (s *SocialService) DeletePost(postID, authorID uint) error {
	post, err := s.PostRepo.FindPostByID(postID)
	if err != nil {
		return e.ErrPostNotFound
	}
	if post.AuthorID != authorID {
		return e.ErrPermission
	}
	post.Status = 2
	if err := s.PostRepo.UpdatePost(post); err != nil {
		return e.ErrServer
	}
	s.DeletePostCache(postID)
	return nil
}
func (s *SocialService) DeletePostCache(postID uint) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf(CacheKeyPostDetail, postID)
	s.rdb.Del(ctx, cacheKey)
}
func (s *SocialService) Search(keyword string, page, pageSize int) ([]model.Post, error) {
	offset := (page - 1) * pageSize
	return s.PostRepo.SearchPosts(keyword, offset, pageSize)
}

// 处理互动逻辑

func (s *SocialService) FollowUser(followerID, followeeID uint) error {
	if followerID == followeeID {
		return e.ErrSelfAction
	}
	err := s.RelationRepo.Follow(followerID, followeeID)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return e.ErrAlreadyFollowing
		}
		s.sendNotification(followeeID, followerID, model.NotifyTypeFollow, "关注了你", 0)
		return e.ErrServer
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in pushPostsToFeed: %v", r)
			}
		}()
		s.pushPostsToFeed(followerID, followeeID)
	}()
	s.sendNotification(followeeID, followerID, model.NotifyTypeFollow, "关注了你", 0)
	return nil
}
func (s *SocialService) UnfollowUser(followerID, followeeID uint) error {
	if err := s.RelationRepo.Unfollow(followerID, followeeID); err != nil {
		return e.ErrServer
	}
	return nil
}

func (s *SocialService) ToggleLike(userID uint, targetID uint, targetType int) error {
	if targetType == 1 {
		post, err := s.PostRepo.FindPostByID(targetID)
		if err != nil {
			return e.ErrPostNotFound
		}
		hasLiked, err := s.LikeRepo.IsLike(userID, targetID, targetType)
		if err != nil {
			return e.ErrServer
		}
		if hasLiked {
			if err := s.LikeRepo.RemoveLike(userID, targetID, post.Type); err != nil {
				return e.ErrServer
			}
			newScore := post.Hotscore - 10
			if newScore < 0 {
				newScore = 0
			}
			return s.PostRepo.UpdateHotScore(targetID, newScore)
		}
		like := &model.Like{UserID: userID, TargetID: targetID, Type: targetType}
		if err := s.LikeRepo.AddLike(like); err != nil {
			return e.ErrServer
		}
		newScore := post.Hotscore + 10
		s.sendNotification(post.AuthorID, userID, model.NotifyTypeLike, "赞了你的文章", targetID)
		return s.PostRepo.UpdateHotScore(targetID, newScore)
	} else if targetID == 2 {
		_, err := s.CommentRepo.FindCommentByID(targetID)
		if err != nil {
			return e.ErrInvalidArgs //error comment not found
		}
		hasLiked, err := s.LikeRepo.IsLike(userID, targetID, targetType)
		if err != nil {
			return e.ErrServer
		}
		if hasLiked {
			return s.LikeRepo.RemoveLike(userID, targetID, targetType)
		}
		like := &model.Like{UserID: userID, TargetID: targetID, Type: targetType}
		return s.LikeRepo.AddLike(like)
	}
	return e.ErrInvalidArgs
}
func (s *SocialService) AddComment(postID, authorID uint, content string) error {
	comment := &model.Comment{
		PostID:   postID,
		AuthorID: authorID,
		Content:  content,
	}
	err := s.CommentRepo.CreateComment(comment)
	if err != nil {
		return e.ErrServer
	}
	post, _ := s.PostRepo.FindPostByID(postID)
	if post != nil {
		newScore := post.Hotscore + 5
		_ = s.PostRepo.UpdateHotScore(postID, newScore)
		s.sendNotification(post.AuthorID, authorID, model.NotifyTypeComment, "评论了你的文章", postID)
	}
	return nil
}

// 获取当前用户的粉丝列表
func (s *SocialService) GetFollowers(userID uint, page, pageSize int) ([]model.User, error) {
	offset := (page - 1) * pageSize
	return s.RelationRepo.GetFollowers(userID, offset, pageSize)
}

// 获取当前用户的关注列表
func (s *SocialService) GetFollowees(userID uint, page, pageSize int) ([]model.User, error) {
	offset := (page - 1) * pageSize
	return s.RelationRepo.GetFollowees(userID, offset, pageSize)
}
func (s *SocialService) GetFeed(userID uint, page, pageSize int) ([]model.Post, error) {
	ctx := context.Background()
	key := fmt.Sprintf("%s%d", FeedKeyPrefix, userID)
	start := int64((page - 1) * pageSize)
	end := start + int64(pageSize) - 1
	postIDs, err := s.rdb.ZRevRange(ctx, key, start, end).Result()
	if err == nil && len(postIDs) > 0 {
		posts, err := s.PostRepo.FindPostsByIDs(postIDs)
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
	followeeIDs, err := s.RelationRepo.GetFolloweeIDs(userID)
	if err != nil {
		return nil, e.ErrServer
	}
	if len(followeeIDs) == 0 {
		return []model.Post{}, nil
	}
	offset := (page - 1) * pageSize
	return s.FeedRepo.GetFeedByUserIDs(followeeIDs, offset, pageSize)
}

// 推送feed
func (s *SocialService) distributePostToFollowers(post *model.Post) {
	ctx := context.Background()
	followerIDs, err := s.RelationRepo.GetFollowerIDs(post.AuthorID)
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
func (s *SocialService) BanUser(id uint) error {
	user, err := s.UserRepo.FindUserByID(id)
	if err != nil {
		return e.ErrUserNotFoundInstance
	}
	if user.Role == 2 {
		return e.ErrPermission
	}
	if err := s.UserRepo.BanUser(id); err != nil {
		return e.ErrServer
	}
	_ = s.SendSystemNotice(id, "已被封禁")
	return nil

}
func (s *SocialService) UnbanUser(targetID uint) error {
	targetUser, err := s.UserRepo.FindUserByID(targetID)
	if err != nil {
		return e.ErrUserNotFoundInstance
	}
	if targetUser.Status == 1 {
		return e.ErrUserNormal
	}
	return s.UserRepo.UnbanUser(targetID)
}

// 排行榜补充
func (s *SocialService) GetLeaderboard(limit int) ([]model.Post, error) {
	return s.PostRepo.GetLeaderboard(limit)
}

// 补充点赞统计
type PostDetailVO struct {
	*model.Post
	LikeCount int64 `json:"like_count"`
}

// 添加收藏关注列表
func (s *SocialService) ToggleConn(userID, postID uint) error {
	_, err := s.PostRepo.FindPostByID(postID)
	if err != nil {
		return e.ErrPostNotFound
	}
	isConn, err := s.ConnectionRepo.IsConn(userID, postID)
	if err != nil {
		return e.ErrServer
	}
	if isConn {
		return s.ConnectionRepo.RemoveConn(userID, postID)
	}
	return s.ConnectionRepo.AddConnection(userID, postID)
}

// 获取收藏列表
func (s *SocialService) GetConn(userID uint, page, pageSze int) ([]model.Post, error) {
	offset := (page - 1) * pageSze
	return s.ConnectionRepo.GetConnByUser(userID, offset, pageSze)
}

// 信息通知
func (s *SocialService) sendNotification(recipientID, actorID uint, nType int, content string, targetID uint) {
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
	_ = s.NotificationRepo.CreateNotification(notification)
}
func (s *SocialService) GetNotifications(userID uint, page, pageSize int) ([]model.Notification, error) {
	offset := (page - 1) * pageSize
	return s.NotificationRepo.GetNotifications(userID, offset, pageSize)
}
func (s *SocialService) GetUnreadCount(userID uint) (int64, error) {
	return s.NotificationRepo.GetUnreadCount(userID)
}
func (s *SocialService) MarkNotificationRead(notificationID, userID uint) error {
	return s.NotificationRepo.MarkAsRead(notificationID, userID)
}
func (s *SocialService) MarkAllNotificationsRead(userID uint) error {
	return s.NotificationRepo.MarkAllAsRead(userID)
}

// 私信通知（但没有系统通知）
func generateSessionID(uid1, uid2 uint) string {
	if uid1 < uid2 {
		return fmt.Sprintf("%d_%d", uid1, uid2)
	}
	return fmt.Sprintf("%d_%d", uid2, uid1)
}
func (s *SocialService) SendMessage(senderID, receiverID uint, content string) error {
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
	if err := s.MessageRepo.CreateMessage(msg); err != nil {
		return e.ErrServer
	}
	s.sendNotification(receiverID, senderID, model.NotifyTypeMessage, "给你发来一条私信", 0)
	return nil
}

// 获取聊天记录
func (s *SocialService) GetChatHistory(userID, peerID uint, page, pageSize int) ([]model.Message, error) {
	sessionID := generateSessionID(userID, peerID)
	offset := (page - 1) * pageSize
	messages, err := s.MessageRepo.GetMessageBySession(sessionID, offset, pageSize)
	if err != nil {
		return nil, e.ErrServer
	}
	_ = s.MessageRepo.MarkMessagesAsRead(sessionID, userID)
	return messages, nil
}

// 获取会话列表
func (s *SocialService) GetConversations(userID uint) ([]model.Message, error) {
	return s.MessageRepo.GetConversations(userID)
}

// total Unread
func (s *SocialService) GetTotalUnread(userID uint) (map[string]int64, error) {
	notifyCount, err := s.NotificationRepo.GetUnreadCount(userID)
	if err != nil {
		notifyCount = 0
	}
	msgCount, err := s.MessageRepo.GetUnreadCountByUser(userID)
	if err != nil {
		msgCount = 0
	}
	return map[string]int64{
		"notification_unread": notifyCount,
		"message_unread":      msgCount,
		"total_unread":        notifyCount + msgCount,
	}, nil
}

// 系统通知
func (s *SocialService) SendSystemNotice(recipientID uint, content string) error {
	notification := &model.Notification{
		RecipientID: recipientID,
		ActorID:     0,
		Type:        model.NotifyTypeSystem,
		Content:     content,
		TargetID:    0,
		IsRead:      false,
	}
	return s.NotificationRepo.CreateNotification(notification)
}

// 新增用户公开信息
type UserProfileVO struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Avatar    string    `json:"avatar"`
	Bio       string    `json:"bio"`
	CreatedAt time.Time `json:"created_at"`
}

// 获取他人公开资料
func (s *SocialService) GetUserProfile(targetID uint) (*UserProfileVO, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("user:profile:%d", targetID)
	val, err := s.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		if val == "NULL" {
			return nil, e.ErrUserNotFoundInstance
		}
		var profile UserProfileVO
		if json.Unmarshal([]byte(val), &profile) == nil {
			return &profile, nil
		}
	}
	user, err := s.UserRepo.FindUserByID(targetID)
	if err != nil {
		s.rdb.Set(ctx, cacheKey, "NULL", time.Minute)
		return nil, e.ErrUserNotFoundInstance
	}
	profile := &UserProfileVO{
		ID:        user.ID,
		Username:  user.Username,
		Avatar:    user.Avatar,
		Bio:       user.Bio,
		CreatedAt: user.CreatedAt,
	}
	data, _ := json.Marshal(profile)
	s.rdb.Set(ctx, cacheKey, data, getRandomExpire(30*time.Minute))
	return profile, nil
}

// 获取文章列表
func (s *SocialService) GetUserPosts(targetID uint, page, pageSize int) ([]model.Post, error) {
	offset := (page - 1) * pageSize
	return s.PostRepo.ListPublicByAuthorID(targetID, offset, pageSize)
}

// 异步将被关注者的文章推送到关注者的时间线
func (s *SocialService) pushPostsToFeed(followerID, followeeID uint) {
	posts, err := s.PostRepo.FindRecentPostIDsByAuthor(followeeID, FeedPushLimit)
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
