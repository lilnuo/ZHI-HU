package model

import "gorm.io/gorm"

// 评论
type Comment struct {
	gorm.Model
	Content  string `Gorm:"type:longtext;not null;comment:评论内容" json:"content"`
	PostID   uint   `Gorm:"not null;index:idx_post;comment:关联的文章或问题的ID" json:"post_id"`
	AuthorID uint   `Gorm:"not null;index:idx_author;comment:评论者ID" json:"author_id"`
	ParentID uint   `Gorm:"default:0;comment:父评论(0表示顶层评论/回答)" json:"parent_id"`

	Author User `Gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	Post   Post `Gorm:"foreignKey:PostID" json:"post,omitempty"`
}

func (Comment) TableName() string {
	return "comments"
}

// 文章
type Post struct {
	gorm.Model
	Title    string `Gorm:"type:varchar(255);not null;comment:标题" json:"title"`
	Content  string `Gorm:"type:longtext;not null;comment:内容(Markdown/HTML)" json:"content"`
	Type     int    `Grom:"type:tinyint;not null;default:1;comment:类型(1:文章,2:问题)" json:"type"`
	AuthorID uint   `Gorm:"not null;index;idx_author;comment:作者ID" json:"authorID"`
	Status   int    `Gorm:"type:tinyint;not null;default:1;comment:状态(0:草稿,1:已发布,2:已删除)" json:"status"`

	Hotscore float64   `Gorm:"type:float;default:0;comment:热度分数" json:"hotscore"`
	Author   User      `Gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	Comments []Comment `Grom:"foreignKey:postID" json:"comments,omitempty"`
}

func (Post) TableName() string {
	return "posts"
}

// 用户
type User struct {
	gorm.Model
	Username string    `Gorm:"type:varchar(32);uniqueIndex;not null;comment:用户名" json:"username"`
	Password string    `Gorm:"type:varchar(128);not null;comment:密码(加盐hash)" json:"-"`
	Email    string    `Gorm:"type:varchar(64);uniqueIndex;comment:邮箱" json:"email"`
	Avatar   string    `Gorm:"type:varchar(255);comment:头像URL" json:"avatar"`
	Role     int       `Gorm:"type:tinyint;default;1;comment:角色(1:普通用户,2:管理员)" json:"role"`
	Status   int       `Gorm:"type:tinyint;default;1;comment;状态(0:禁言,1:正常)" json:"status"`
	Posts    []Post    `Gorm:"foreignKey:AuthorID" json:"posts,omitempty"`
	Comments []Comment `Gorm:"foreignKey:AuthorID" json:"comments,omitempty"`
}

func (User) TableName() string {
	return "users"
}

// 用户关系
type Relation struct {
	gorm.Model
	FollowerID uint `Gorm:"not null;index:idx_follower;comment:粉丝ID" json:"follower_id"`
	FolloweeID uint `Gorm:"not null;index:idx_followee;comment:被关注者ID" json:"followee_id"`
	Follower   User `Gorm:"foreignKey:FollowerID" json:"follower,omitempty"`
	Followee   User `Gorm:"foreignKey:FolloweeID" json:"followee,omitempty"`
}

func (Relation) TableName() string {
	return "relations"
}

// 点赞
type Like struct {
	gorm.Model
	UserID   uint `Gorm:"not null;index:idx_user;comment:用户ID" json:"user_id"`
	TargetID uint `Gorm:"not null;index:idx_target;comment:目标对象ID(文章ID或评论ID)" json:"target_id"`
	Type     int  `Gorm:"type:tinyint;not null;comment:类型(1:点赞,2:收藏)" json:"type"`

	User User `Gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Like) TableName() string {
	return "likes"
}
