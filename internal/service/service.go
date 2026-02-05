package service

import (
	"errors"
	"go-zhihu/internal/model"
	"go-zhihu/internal/repository"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var JwtSecret = []byte("hahaha")

type SocialService struct {
	RelationRepo *repository.RelationRepository
	LikeRepo     *repository.LikeRepository
	PostRepo     *repository.PostRepository
	FeedRepo     *repository.FeedRepository
	CommentRepo  *repository.CommentRepository
	UserRepo     *repository.UserRepository
}

func NewUserService(relationRepo *repository.RelationRepository,
	likeRepo *repository.LikeRepository, postRepo *repository.PostRepository,
	feedRepo *repository.FeedRepository, commentRepo *repository.CommentRepository,
	userRepo *repository.UserRepository) *SocialService {
	return &SocialService{
		RelationRepo: relationRepo,
		LikeRepo:     likeRepo,
		PostRepo:     postRepo,
		FeedRepo:     feedRepo,
		CommentRepo:  commentRepo,
		UserRepo:     userRepo,
	}
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  *model.User `json:"user"`
}

func (s *SocialService) Register(username, password, email string) error {
	_, err := s.UserRepo.FindUsername(username)
	if err == nil {
		return errors.New("already exist")
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user := &model.User{
		Username: username,
		Password: string(hashedPassword),
		Email:    email,
		Role:     1,
		Status:   1, //默认正常
	}
	return s.UserRepo.CreateUser(user)
}

// 鉴权加密，环境获取
func (s *SocialService) generateToken(userID uint, username string) (string, error) {
	claims := &jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JwtSecret)
}
func (s *SocialService) Login(username, password string) (*LoginResponse, error) {
	user, err := s.UserRepo.FindUsername(username)
	if err != nil {
		return nil, errors.New("user not exist")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("password wrong")
	}
	if user.Status == 0 {
		return nil, errors.New("账号已被禁言")
	}
	token, err := s.generateToken(user.ID, user.Username)
	if err != nil {
		return nil, err
	}
	return &LoginResponse{
		Token: token,
		User:  user,
	}, nil
}

// 处理内容的发布、更新、获取和删

func (s *SocialService) CreatePost(authorID uint, title, content string, postType int) error {
	post := &model.Post{
		Title:    title,
		Content:  content,
		Type:     postType, //1.chapter,2.question
		AuthorID: authorID,
		Status:   1, //默认发布
		Hotscore: 0,
	}
	return s.PostRepo.CreatePost(post)
}
func (s *SocialService) GetPostDetail(postID uint) (*model.Post, error) {
	return s.PostRepo.FindByID(postID)
}
func (s *SocialService) UpdatePost(postID, authorID uint, title, content string) error {
	post, err := s.PostRepo.FindByID(postID)
	if err != nil {
		return nil
	}
	if post.AuthorID != authorID {
		return errors.New("无权修改此文章")
	}
	post.Title = title
	post.Content = content
	return s.PostRepo.UpdatePost(post)
}
func (s *SocialService) DeletePost(postID, authorID uint) error {
	post, err := s.PostRepo.FindByID(postID)
	if err != nil {
		return err
	}
	if post.AuthorID != authorID {
		return errors.New("无权删除")
	}
	post.Status = 2
	return s.PostRepo.UpdatePost(post)
}
func (s *SocialService) Search(keyword string, page, pageSize int) ([]model.Post, error) {
	offset := (page - 1) * pageSize
	return s.PostRepo.SearchPosts(keyword, offset, pageSize)
}

// 处理互动逻辑

func (s *SocialService) FollowUser(followerID, followeeID uint) error {
	if followerID == followeeID {
		return errors.New("不能关注自己")
	}
	isFollowing, _ := s.RelationRepo.IsFollowing(followerID, followeeID)
	if isFollowing {
		return errors.New("已经关注了")
	}
	return s.RelationRepo.Follow(followerID, followeeID)
}
func (s *SocialService) UnfollowUser(followerID, followeeID uint) error {
	return s.RelationRepo.Unfollow(followerID, followeeID)
}
func (s *SocialService) ToggleLike(userID uint, targetID uint) error {
	post, err := s.PostRepo.FindByID(targetID)
	if err != nil {
		return errors.New("not exist")
	}
	hasLiked, err := s.LikeRepo.IsLike(userID, targetID)
	if err != nil {
		return err
	}
	if hasLiked {
		if err := s.LikeRepo.RemoveLike(userID, targetID, post.Type); err != nil {
			return err
		}
		newScore := post.Hotscore - 10
		if newScore < 0 {
			newScore = 0
		}
		return s.PostRepo.UpdateHotScore(targetID, newScore)
	}
	like := &model.Like{UserID: userID, TargetID: targetID}
	if err := s.LikeRepo.AddLike(like); err != nil {
		return err
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
		return err
	}
	post, _ := s.PostRepo.FindByID(postID)
	if post != nil {
		newScore := post.Hotscore + 5
		_ = s.PostRepo.UpdateHotScore(postID, newScore)
	}
	return nil
}
func (s *SocialService) GetFeed(userID uint, page, pageSize int) ([]model.Post, error) {
	followeeIDs, err := s.RelationRepo.GetFolloweeIDs(userID)
	if err != nil {
		return nil, err
	}
	if len(followeeIDs) == 0 {
		return []model.Post{}, nil
	}
	offset := (page - 1) * pageSize
	posts, err := s.FeedRepo.GetFeedByUserIDs(followeeIDs, offset, pageSize)
	if err != nil {
		return nil, err
	}
	return posts, nil
}
func (s *SocialService) BanUser(id uint) error {
	isBanned, err := s.UserRepo.IsUserBanned(id)
	if err != nil {
		return errors.New("用户不存在或数据库查询失败")
	}
	user, err := s.UserRepo.FindByID(id)
	if err != nil {
		return err
	}
	if user.Role == 1 {
		return errors.New("无权禁言管理员")
	}
	if !isBanned {
		return errors.New("正常，无需解除管理")
	}
	return s.UserRepo.BanUser(id)
}
func (s *SocialService) UnbanUser(targetID uint) error {
	targetUser, err := s.UserRepo.FindByID(targetID)
	if err != nil {
		return errors.New("用户不存在")
	}
	if targetUser.Status == 1 {
		return errors.New("正常")
	}
	return s.UserRepo.UnbanUser(targetID)
}
