package middleware

import (
	"context"
	"fmt"
	"go-zhihu/config"
	"go-zhihu/internal/repository"
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"

	"net/http"
	"strings"
)

type Claims struct {
	ID       uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "需要登录"})
			return
		}
		tokenString := authHeader
		if !strings.HasPrefix(tokenString, "Bearer ") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Token 格式错误"})
			return
		}
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.Setting.JWT.Secret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token invalid"})
			c.Abort()
			return
		}
		if claims, ok := token.Claims.(*Claims); ok {
			c.Set("user_id", claims.ID)
			c.Set("username", claims.Username)
		}
		c.Next()
	}
}

func RateLimit(rdb *redis.Client, requestLimit int) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}
		key := fmt.Sprintf("rate_limit:user:%v", userID)
		ctx := context.Background()
		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}
		if count == 1 {
			rdb.Expire(ctx, key, time.Minute)
		}
		if count > int64(requestLimit) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "操作too 频繁"})
			c.Abort()
			return
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(requestLimit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(int(requestLimit)-int(count)))
		c.Next()
	}
}
func CheckStatus(userRepo *repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户身份未确认"})
			c.Abort()
			return
		}
		uid := userID.(uint)
		isBanned, err := userRepo.IsUserBanned(uid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法验证用户状态"})
			c.Abort()
			return
		}
		if isBanned {
			c.JSON(http.StatusForbidden, gin.H{"error": "账号已被禁言，无法进行操作"})
			c.Abort()
			return
		}
		c.Next()
	}
}
func CustomRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[panic recovered]URI:%s|Error:%v\n", c.Request.URL.Path, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"code": 500,
					"msg":  "服务器内部错误",
					"data": nil,
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
