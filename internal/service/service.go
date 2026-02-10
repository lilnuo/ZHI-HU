package service

import (
	"go-zhihu/config"
	"go-zhihu/internal/model"
	"go-zhihu/internal/repository"
	"go-zhihu/pkg/e"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type SocialService struct {
	RelationRepo   *repository.RelationRepository
	LikeRepo       *repository.LikeRepository
	PostRepo       *repository.PostRepository
	FeedRepo       *repository.FeedRepository
	CommentRepo    *repository.CommentRepository
	UserRepo       *repository.UserRepository
	ConnectionRepo *repository.ConnectRepository
	rdb            *redis.Client
	secret         string
}

func NewUserService(relationRepo *repository.RelationRepository,
	likeRepo *repository.LikeRepository, postRepo *repository.PostRepository,
	feedRepo *repository.FeedRepository, commentRepo *repository.CommentRepository,
	userRepo *repository.UserRepository,
	rdb *redis.Client, jwtSecret string,
	connectionRepo *repository.ConnectRepository) *SocialService {
	return &SocialService{
		RelationRepo:   relationRepo,
		LikeRepo:       likeRepo,
		PostRepo:       postRepo,
		FeedRepo:       feedRepo,
		CommentRepo:    commentRepo,
		UserRepo:       userRepo,
		ConnectionRepo: connectionRepo,
		rdb:            rdb,
		secret:         jwtSecret,
	}
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
	_, err := s.UserRepo.FindUserByID(userID)
	if err != nil {
		return e.ErrUserNotFoundInstance
	}
	if len(bio) > 500 {
		return e.ErrInvalidArgs
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
	post, err := s.PostRepo.FindPostByID(postID)
	if err != nil {
		return nil, e.ErrPostNotFound
	}
	count, err := s.LikeRepo.CountLikes(postID)
	if err != nil {
		count = 0
	}
	return &PostDetailVO{
		Post:      post,
		LikeCount: count,
	}, nil
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
	return nil
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
	isFollowing, _ := s.RelationRepo.IsFollowing(followerID, followeeID)
	if isFollowing {
		return e.ErrAlreadyFollowing
	}
	if err := s.RelationRepo.Follow(followerID, followeeID); err != nil {
		return e.ErrServer
	}
	return nil
}
func (s *SocialService) UnfollowUser(followerID, followeeID uint) error {
	if err := s.RelationRepo.Unfollow(followerID, followeeID); err != nil {
		return e.ErrServer
	}
	return nil
}

func (s *SocialService) ToggleLike(userID uint, targetID uint, targetType int) error {
	post, err := s.PostRepo.FindPostByID(targetID)
	if err != nil {
		return e.ErrPostNotFound
	}
	targetType = post.Type
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
	return s.PostRepo.UpdateHotScore(targetID, newScore)
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
	followeeIDs, err := s.RelationRepo.GetFolloweeIDs(userID)
	if err != nil {
		return nil, e.ErrServer
	}
	if len(followeeIDs) == 0 {
		return []model.Post{}, nil
	}
	offset := (page - 1) * pageSize
	posts, err := s.FeedRepo.GetFeedByUserIDs(followeeIDs, offset, pageSize)
	if err != nil {
		return nil, e.ErrServer
	}
	return posts, nil
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

// 获取列表
func (s *SocialService) GetConn(userID uint, page, pageSze int) ([]model.Post, error) {
	offset := (page - 1) * pageSze
	return s.ConnectionRepo.GetConnByUser(userID, offset, pageSze)
}
