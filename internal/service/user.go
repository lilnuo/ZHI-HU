package service

import (
	"context"
	"encoding/json"
	"fmt"
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

func (s *UserService) Register(username, password, email string) error {
	_, err := s.repo.FindUsername(username)
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
	if err := s.repo.CreateUser(user); err != nil {
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
func (s *UserService) Login(username, password string) (*LoginResponse, error) {
	user, err := s.repo.FindUsername(username)
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
func (s *UserService) UpdateProfile(userID uint, avatar, bio string) error {
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
	if err := s.repo.UpdateProfile(userID, avatar, bio); err != nil {
		return e.ErrServer
	}
	return nil
}
func (s *UserService) BanUser(id uint) error {
	user, err := s.repo.FindUserByID(id)
	if err != nil {
		return e.ErrUserNotFoundInstance
	}
	if user.Role == 2 {
		return e.ErrPermission
	}
	if err := s.repo.BanUser(id); err != nil {
		return e.ErrServer
	}
	_ = s.notify.SendSystemNotice(id, "已被封禁")
	return nil

}
func (s *UserService) UnbanUser(targetID uint) error {
	targetUser, err := s.repo.FindUserByID(targetID)
	if err != nil {
		return e.ErrUserNotFoundInstance
	}
	if targetUser.Status == 1 {
		return e.ErrUserNormal
	}
	return s.repo.UnbanUser(targetID)
}

// 获取他人公开资料
func (s *UserService) GetUserProfile(targetID uint) (*UserProfileVO, error) {
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
	user, err := s.repo.FindUserByID(targetID)
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
