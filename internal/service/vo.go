package service

import (
	"go-zhihu/internal/model"
	"time"
)

// 补充点赞统计
type PostDetailVO struct {
	*model.Post
	LikeCount int64 `json:"like_count"`
}

// 新增用户公开信息
type UserProfileVO struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Avatar    string    `json:"avatar"`
	Bio       string    `json:"bio"`
	CreatedAt time.Time `json:"created_at"`
}
