package service

import (
	"context"
	"encoding/json"
	"fmt"
	"go-zhihu/config"
	"go-zhihu/internal/model"
	"go-zhihu/internal/repository"
	"go-zhihu/pkg/e"
	"log"
	"net/mail"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	repo   *repository.UserRepository
	notify *NotificationService
	rdb    *redis.Client
	secret string
}

func NewUserService(repo *repository.UserRepository, notify *NotificationService, secret string) *UserService {
	return &UserService{repo: repo, notify: notify, secret: secret}
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  *model.User `json:"user"`
}

func (s *UserService) Register(ctx context.Context, tx *gorm.DB, username, password, email string) error {
	if username == "" || password == "" || email == "" {
		return e.ErrInvalidArgs
	}
	if utf8.RuneCountInString(username) < 3 || utf8.RuneCountInString(username) > 32 {
		return e.ErrInvalidArgs
	}
	if len(password) < 6 {
		return e.ErrInvalidArgs
	}
	_, err := mail.ParseAddress(email)
	if err != nil {
		return e.ErrInvalidArgs
	}
	_, err = s.repo.FindUsername(ctx, tx, username)
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
	if err := s.repo.CreateUser(ctx, tx, user); err != nil {
		return e.ErrServer
	}
	return nil
}

// 鉴权加密，环境获取
func (s *UserService) generateToken(userID uint, username string, role int) (string, error) {
	claims := &jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"role":     strconv.Itoa(role),
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.Setting.JWT.Secret))
}
func (s *UserService) Login(ctx context.Context, tx *gorm.DB, username, password string) (*LoginResponse, error) {
	user, err := s.repo.FindUsername(ctx, tx, username)
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
func (s *UserService) UpdateProfile(ctx context.Context, tx *gorm.DB, userID uint, avatar, bio string) error {
	if err := s.repo.UpdateProfile(ctx, tx, userID, avatar, bio); err != nil {
		return e.ErrServer
	}
	cacheKey := fmt.Sprintf("user:profile:%d", userID)
	if err := s.rdb.Del(ctx, cacheKey).Err(); err != nil {
		log.Printf("failed to invalidate user cache :%v", err)
	}
	return nil
}
func (s *UserService) BanUser(ctx context.Context, tx *gorm.DB, id uint) error {
	user, err := s.repo.FindUserByID(ctx, tx, id)
	if err != nil {
		return e.ErrUserNotFoundInstance
	}
	if user.Role == 2 {
		return e.ErrPermission
	}
	if err := s.repo.BanUser(ctx, tx, id); err != nil {
		return e.ErrServer
	}
	_ = s.notify.SendSystemNotice(ctx, tx, id, "已被封禁")
	return nil

}
func (s *UserService) UnbanUser(ctx context.Context, tx *gorm.DB, targetID uint) error {
	targetUser, err := s.repo.FindUserByID(ctx, tx, targetID)
	if err != nil {
		return e.ErrUserNotFoundInstance
	}
	if targetUser.Status == 1 {
		return e.ErrUserNormal
	}
	return s.repo.UnbanUser(ctx, tx, targetID)
}

// 获取他人公开资料
func (s *UserService) GetUserProfile(ct context.Context, tx *gorm.DB, targetID uint) (*UserProfileVO, error) {
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
	user, err := s.repo.FindUserByID(ct, tx, targetID)
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
