package model

import (
	"time"

	"gorm.io/gorm"
)

// 评论
type Comment struct {
	gorm.Model
	Content  string `gorm:"type:longtext;not null;comment:评论内容" json:"content"`
	PostID   uint   `gorm:"not null;index:idx_post;comment:关联的文章或问题的ID" json:"post_id"`
	AuthorID uint   `gorm:"not null;index:idx_author;comment:评论者ID" json:"author_id"`
	ParentID uint   `gorm:"default:0;comment:父评论(0表示顶层评论/回答)" json:"parent_id"`

	Author User `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	Post   Post `gorm:"foreignKey:PostID" json:"post,omitempty"`
}

// 文章
type Post struct {
	gorm.Model
	Title    string `gorm:"type:varchar(255);not null;comment:标题" json:"title"`
	Content  string `gorm:"type:longtext;not null;comment:内容(Markdown/HTML)" json:"content"`
	Type     int    `grom:"type:tinyint;not null;default:1;comment:类型(1:文章,2:问题)" json:"type"`
	AuthorID uint   `gorm:"not null;index:idx_author;comment:作者ID" json:"authorID"`
	Status   int    `gorm:"type:tinyint;not null;default:1;comment:状态(0:草稿,1:已发布,2:已删除)" json:"status"`

	Hotscore float64   `gorm:"type:float;default:0;comment:热度分数" json:"hot_score"`
	Author   User      `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	Comments []Comment `gorm:"foreignKey:postID" json:"comments,omitempty"`
}

// 用户
type User struct {
	gorm.Model
	Username string    `gorm:"type:varchar(32);uniqueIndex;not null;comment:用户名" json:"username"`
	Password string    `gorm:"type:varchar(128);not null;comment:密码(加盐hash)" json:"-"`
	Email    string    `gorm:"type:varchar(64);uniqueIndex;comment:邮箱" json:"email"`
	Avatar   string    `gorm:"type:varchar(255);comment:头像URL" json:"avatar"`
	Bio      string    `gorm:"varchar(255);comment:头像URL" json:"bio"`
	Role     int       `gorm:"type:tinyint;default;1;comment:角色(1:普通用户,2:管理员)" json:"role"`
	Status   int       `gorm:"type:tinyint;default;1;comment;状态(0:禁言,1:正常)" json:"status"`
	Posts    []Post    `gorm:"foreignKey:AuthorID" json:"posts,omitempty"`
	Comments []Comment `gorm:"foreignKey:AuthorID" json:"comments,omitempty"`
}

// 用户关系
type Relation struct {
	gorm.Model
	FollowerID uint `gorm:"not null;index:idx_follower;comment:粉丝ID" json:"follower_id"`
	FolloweeID uint `gorm:"not null;index:idx_followee;comment:被关注者ID" json:"followee_id"`
	Follower   User `gorm:"foreignKey:FollowerID" json:"follower,omitempty"`
	Followee   User `gorm:"foreignKey:FolloweeID" json:"followee,omitempty"`
}

// 点赞
type Like struct {
	gorm.Model
	UserID   uint `gorm:"not null;index:idx_user;comment:用户ID" json:"user_id"`
	TargetID uint `gorm:"not null;index:idx_target;comment:目标对象ID(文章ID或评论ID)" json:"target_id"`
	Type     int  `gorm:"type:tinyint;not null;comment:类型(1:问题/文章,2:评论)" json:"type"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// 收藏
type Connection struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"index;not null"`
	PostID    uint      `gorm:"index;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// 通知模型
type Notification struct {
	gorm.Model
	RecipientID uint   `gorm:"not null;index;comment:接受者ID" json:"recipient_id"`
	ActorID     uint   `gorm:"not null;comment:触发者ID" json:"actor_id"`
	Type        int    `gorm:"not null;comment:类型(1:点赞,2:评论,3:关注,4:系统)" json:"type"`
	Content     string `gorm:"type:varchar(255);comment:通知内容" json:"content"`
	TargetID    uint   `gorm:"comment:关联对象ID(如文章ID)" json:"target_id"`
	IsRead      bool   `gorm:"default:false;comment:是否已读" json:"is_read"`
	Actor       User   `gorm:"foreignKey:ActorID" json:"actor"`
	Recipient   User   `gorm:"foreignKey:RecipientID" json:"-"`
}

const (
	NotifyTypeLike    = 1
	NotifyTypeComment = 2
	NotifyTypeFollow  = 3
	NotifyTypeSystem  = 4
	NotifyType        = 5
	NotifyTypeMessage = 6
)
const (
	PostStatusDraft     = 0
	PostStatusPublished = 1
	PostStatusDeleted   = 2
)
const (
	TargetTypePost    = 1
	TargetTypeComment = 2
)

// 私信模型
type Message struct {
	gorm.Model
	SenderID   uint   `gorm:"not null;index;comment:发送者ID" json:"sender_id"`
	ReceiverID uint   `gorm:"not null;index;comment:接受者ID" json:"receiver_id"`
	Content    string `gorm:"type:text;not null;comment:消息内容" json:"content"`
	Session    string `gorm:"index;not null;comment:会话ID(用于分组)" json:"session_id"`
	IsRead     bool   `gorm:"default:false;comment:是否已读" json:"is_read"`

	Sender   User `gorm:"foreignKey:SenderID" json:"sender"`
	Receiver User `gorm:"foreignKey:ReceiverID" json:"receiver"`
}
