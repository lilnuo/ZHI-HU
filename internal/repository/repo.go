package repository

import (
	"errors"
	"go-zhihu/internal/model"
	"go-zhihu/pkg/e"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

// 用户查找函数&&状态更新
func (r *UserRepository) CreateUser(user *model.User) error {
	return r.DB.Create(user).Error
}
func (r *UserRepository) FindUsername(username string) (*model.User, error) {
	var user model.User
	err := r.DB.Where("username=?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
func (r *UserRepository) FindUserByID(id uint) (*model.User, error) {
	var user model.User
	err := r.DB.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// 个人信息更改
func (r *UserRepository) UpdateProfile(userID uint, avatar, bio string) error {
	updates := map[string]interface{}{}
	if avatar != "" {
		updates["avatar"] = avatar
	}
	if bio != "" {
		updates["bio"] = bio
	}
	if len(updates) == 0 {
		return nil
	}
	return r.DB.Model(&model.User{}).Where("id=?", userID).Updates(updates).Error
}

type PostRepository struct {
	DB *gorm.DB
}

func NewPostRepository(db *gorm.DB) *PostRepository {
	return &PostRepository{DB: db}
}

// 处理文章操作：发布、更新、获取详细、热度排序、搜
func (r *PostRepository) CreatePost(post *model.Post) error {
	return r.DB.Create(post).Error
}
func (r *PostRepository) FindPostByID(id uint) (*model.Post, error) {
	var post model.Post
	err := r.DB.Where("status IN ?", []int{0, 1}).Preload("Author").First(&post, id).Error
	if err != nil {
		return nil, e.ErrInvalidArgs
	}
	if post.Status == 2 {
		return nil, e.ErrPostNotFound
	}
	return &post, err
}
func (r *PostRepository) FindPostsByIDs(ids []string) ([]model.Post, error) {
	var posts []model.Post
	err := r.DB.Where("id IN ?", ids).Where("status = ?", 1).Preload("Author").Find(&posts).Error
	return posts, err
}
func (r *PostRepository) UpdateStatus(postID uint, status int) error {
	return r.DB.Model(&model.Post{}).Where("id = ?", postID).Update("status", status).Error
}
func (r *PostRepository) UpdatePost(post *model.Post) error {
	return r.DB.Save(post).Error
}
func (r *PostRepository) UpdateHotScore(targetID uint, newScore float64) error {
	return r.DB.Model(&model.Post{}).Where("id=?", targetID).Update("hot_score", newScore).Error
}
func (r *PostRepository) ListPosts(offset, limit int, orderBy string) ([]model.Post, error) {
	var posts []model.Post
	err := r.DB.Where("status=?", 1).Preload("Author").Order(orderBy + " desc").Offset(offset).Limit(limit).Find(&posts).Error
	return posts, err
}

// 获取指定用户主页
func (r *PostRepository) ListPublicByAuthorID(authorID uint, offset, limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.DB.Where("author_id = ? AND status = ?", authorID, 1).Preload("Author").Order("created_at DESC").Offset(offset).Limit(limit).Find(&posts).Error
	return posts, err
}

// 获取指定用户草稿箱列表
func (r *PostRepository) ListDrafts(userID uint, offset, limit int) ([]model.Post, error) {
	var posts []model.Post
	//降序排列
	err := r.DB.Where("author_id = ? AND status=?", userID, 0).Order("updated_at DESC").Offset(offset).Limit(limit).Find(&posts).Error
	return posts, err
}

// 检查空字符，mysql的match语法不支持空字符
func cleanFullTextKeyword(keyword string) string {
	reg := regexp.MustCompile(`[^\p{Han}a-zA-Z0-9\s]`)
	return reg.ReplaceAllString(keyword, " ")
}
func (r *PostRepository) SearchPosts(keyword string, offset, limit int) ([]model.Post, error) {
	var posts []model.Post
	if strings.TrimSpace(keyword) == "" {
		return []model.Post{}, nil
	}
	processedKeyword := cleanFullTextKeyword(keyword)
	if strings.TrimSpace(processedKeyword) == "" {
		return []model.Post{}, nil
	}
	err := r.DB.Model(&model.Post{}).Preload("Author").Where("status=?", 1).Where("MATCH(title,content) AGAINST(? IN NATURAL LANGUAGE MODE)", keyword).Order("created_at DESC").Offset(offset).Limit(limit).Find(&posts).Error
	return posts, err
}

// 获取指定用户最近的文章id和创建时间
func (r *PostRepository) FindRecentPostIDsByAuthor(authorID uint, limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.DB.Model(&model.Post{}).Select("id,created_at").Where("author_id = ? AND status = ?", authorID, 1).Order("created_at DESC").Limit(limit).Find(&posts).Error
	return posts, err
}

type CommentRepository struct {
	DB *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{DB: db}
}

// 处理评论和回答的发布与对话
func (r *CommentRepository) CreateComment(comment *model.Comment) error {
	return r.DB.Create(comment).Error
}
func (r *CommentRepository) GetCommentByPostID(postID uint) ([]model.Comment, error) {
	var comments []model.Comment
	err := r.DB.Where("post_id=?", postID).Preload("Author").Order("created_at asc").Find(&comments).Error
	return comments, err
}

type RelationRepository struct {
	DB *gorm.DB
}

func NewRelationRepository(db *gorm.DB) *RelationRepository {
	return &RelationRepository{DB: db}
}

// 处理关注关系
func (r *RelationRepository) Follow(followerID, followeeID uint) error {
	relation := model.Relation{
		FollowerID: followerID,
		FolloweeID: followeeID,
	}
	return r.DB.Create(&relation).Error
}

func (r *RelationRepository) Unfollow(followerID, followeeID uint) error {
	return r.DB.Where("follower_id=? AND followee_id=?", followerID, followeeID).Delete(&model.Relation{}).Error
}
func (r *RelationRepository) GetFolloweeIDs(userID uint) ([]uint, error) {
	var ids []uint
	err := r.DB.Model(&model.Relation{}).Where("follower_id=?", userID).Pluck("followee_id", &ids).Error
	return ids, err
}
func (r *RelationRepository) GetFollowerIDs(userID uint) ([]uint, error) {
	var ids []uint
	err := r.DB.Model(&model.Relation{}).Where("followee_id=?", userID).Pluck("follower_id", &ids).Error
	return ids, err
}
func (r *RelationRepository) IsFollowing(followerID, followeeID uint) (bool, error) {
	var count int64
	err := r.DB.Model(&model.Relation{}).Where("follower_id=? AND followee_id=?", followerID, followeeID).Count(&count).Error
	return count > 0, err
}

// 获取粉丝列表（谁关注了我）
func (r *RelationRepository) GetFollowers(userID uint, offset, limit int) ([]model.User, error) {
	var users []model.User
	err := r.DB.Table("users").Select("users.id,users.username,users.avatar,users.created_at").Joins("JOIN relations ON user_id = relations.follower_id").Where("relations.followee_id=?", userID).Order("relations.created_at DESC").Offset(offset).Limit(limit).Find(&users).Error
	return users, err
}

// 获取关注列表
func (r *RelationRepository) GetFollowees(userID uint, offset, limit int) ([]model.User, error) {
	var users []model.User
	err := r.DB.Table("users").Select("users.id,users.username,users.avatar,users.created_at").Joins("JOIN relations ON users.id = relations.followee_id").Where("relations.follower_id =?", userID).Order("relations.created_at DESC").Offset(offset).Limit(limit).Find(&users).Error
	return users, err
}

type LikeRepository struct {
	DB *gorm.DB
}

func NewLikeRepository(db *gorm.DB) *LikeRepository {
	return &LikeRepository{DB: db}
}

// 处理点赞
func (r *LikeRepository) AddLike(like *model.Like) error {
	return r.DB.Create(like).Error
}
func (r *LikeRepository) RemoveLike(userID, targetID uint, likeType int) error {
	return r.DB.Where("user_id=? AND target_id=? AND type=?", userID, targetID, likeType).Delete(&model.Like{}).Error
}
func (r *LikeRepository) CountLikes(targetID uint) (int64, error) {
	var count int64
	err := r.DB.Model(&model.Like{}).Where("target_id=? AND type=1", targetID).Count(&count).Error
	return count, err
}
func (r *LikeRepository) IsLike(userID, targetID uint, targetType int) (bool, error) {
	var count int64
	err := r.DB.Model(&model.Like{}).Where("user_id=? AND target_id=? AND type=?", userID, targetID, targetType).Count(&count).Error
	return count > 0, err
}

// 事务方法
func (r *LikeRepository) ToggleLikeWithTx(userID, targetID uint, targetType int) (bool, bool, error) {
	var isLike bool
	var isNewAction bool
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		var existingLike model.Like
		err := tx.Where("user_id = ? AND target_id = ? AND type =?", userID, targetID, targetType).First(existingLike).Error

		const likePostScore = 10
		const likeCommentScore = 5
		if err == nil {
			isNewAction = false
			isLike = false
			if err := tx.Delete(&existingLike).Error; err != nil {
				return err
			}
			if targetType == 1 {
				if err := tx.Model(&model.Post{}).Where("id = ?", targetID).Update("hot_score", gorm.Expr("hot_score - ?", likePostScore)).Error; err != nil {
					return err
				}
			} else if targetType == 2 {
				var comment model.Comment
				if err := tx.Select("post_id").Where("id = ?", targetID).First(&comment).Error; err != nil {
					return err
				}
			}
			return nil
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			isNewAction = true
			isLike = true
			like := &model.Like{UserID: userID, TargetID: targetID, Type: targetType}
			if err := tx.Create(like).Error; err != nil {
				return err
			}
			if targetType == 1 {
				if err := tx.Model(&model.Post{}).Where("id = ?", targetID).Update("hot_score + ?", likePostScore).Error; err != nil {
					return err
				}
			} else if targetType == 2 {
				var comment model.Comment
				if err := tx.Select("post_id").Where("id = ?", targetID).First(&comment).Error; err != nil {
					return nil
				}
				if err := tx.Model(&model.Post{}).Where("id = ?", comment.PostID).Update("hot_score", gorm.Expr("hot_score+ ?", likeCommentScore)).Error; err != nil {
					return nil
				}
			}
			return nil
		}
		return err

	})
	return isLike, isNewAction, err
}

type FeedRepository struct {
	DB *gorm.DB
}

func NewFeedRepository(db *gorm.DB) *FeedRepository {
	return &FeedRepository{DB: db}
}

// 获取用户动态

func (r *FeedRepository) GetFeedByUserIDs(userIDs []uint, offset, limit int) ([]model.Post, error) {
	if len(userIDs) == 0 {
		return []model.Post{}, nil
	}
	var posts []model.Post
	err := r.DB.Where("author_id IN ?", userIDs).Where("status=1").Preload("Author").Order("created_at DESC").Offset(offset).Limit(limit).Find(&posts).Error
	return posts, err
}

// 禁言处理补充
func (r *UserRepository) BanUser(id uint) error {
	return r.DB.Model(&model.User{}).Where("id=?", id).Update("status", 0).Error
}
func (r *UserRepository) UnbanUser(id uint) error {
	return r.DB.Model(&model.User{}).Where("id=?", id).Update("status", 1).Error
}
func (r *UserRepository) IsUserBanned(id uint) (bool, error) {
	var user model.User
	err := r.DB.Select("status").First(&user, id).Error
	if err != nil {
		return false, err
	}
	return user.Status == 0, nil
}

// 排行榜补充
func (r *PostRepository) GetLeaderboard(limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.DB.Where("status=?", 1).Preload("Author").Order("hot_score DESC").Limit(limit).Find(&posts).Error
	return posts, err
}

type ConnectRepository struct {
	DB *gorm.DB
}

func NewConnectionRepository(db *gorm.DB) *ConnectRepository {
	return &ConnectRepository{DB: db}
}

// 关注文章或问题
func (r ConnectRepository) AddConnection(userID, postID uint) error {
	conn := &model.Connection{
		UserID: userID,
		PostID: postID,
	}
	return r.DB.Create(conn).Error
}
func (r *ConnectRepository) RemoveConn(userID, postID uint) error {
	return r.DB.Where("user_id=? AND post_id =?", userID, postID).Delete(&model.Connection{}).Error
}

func (r *ConnectRepository) IsConn(userID, postID uint) (bool, error) {
	var count int64
	err := r.DB.Model(&model.Connection{}).Where("user_id=? AND post_id=?", userID, postID).Count(&count).Error
	return count > 0, err
}

// 获取关注收藏列表
func (r *ConnectRepository) GetConnByUser(userID uint, offset, limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.DB.Table("posts").Joins("JOIN connections ON posts.id =connections.post_id").Where("connections.user_id=?", userID).Order("connections.created_at DESC").Offset(offset).Limit(limit).Error
	return posts, err
}
func (r *CommentRepository) FindCommentByID(id uint) (*model.Comment, error) {
	var comment model.Comment
	err := r.DB.First(&comment, id).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// 信息通知
type NotificationRepository struct {
	DB *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{DB: db}
}
func (r *NotificationRepository) CreateNotification(n *model.Notification) error {
	return r.DB.Create(n).Error
}
func (r *NotificationRepository) GetNotifications(userID uint, offset, limit int) ([]model.Notification, error) {
	var notifications []model.Notification
	err := r.DB.Where("recipient_id =?", userID).Preload("Actor").Order("created_at DESC").Offset(offset).Limit(limit).Find(&model.Notification{}).Error
	return notifications, err
}

// 红标信息
func (r *NotificationRepository) GetUnreadCount(userID uint) (int64, error) {
	var count int64
	err := r.DB.Model(&model.Notification{}).Where("recipient_id = ? AND is_read=?", userID, false).Count(&count).Error
	return count, err
}
func (r *NotificationRepository) MarkAsRead(notificationID, userID uint) error {
	return r.DB.Model(&model.Notification{}).Where("id =? AND recipient_id =?", notificationID, userID).Update("is_read", true).Error
}
func (r *NotificationRepository) MarkAllAsRead(userID uint) error {
	return r.DB.Model(&model.Notification{}).Where("recipient_id =? AND is_read= ?", userID, false).Update("is_read", true).Error
}
func (r *NotificationRepository) DeleteNotification(id uint) error {
	return r.DB.Delete(&model.Notification{}, id).Error
}

// 关注私信
type MessageRepository struct {
	DB *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{DB: db}
}
func (r *MessageRepository) CreateMessage(msg *model.Message) error {
	return r.DB.Create(msg).Error
}
func (r *MessageRepository) GetMessageBySession(sessionID string, offset, limit int) ([]model.Message, error) {
	var messages []model.Message
	err := r.DB.Where("session_id = ?", sessionID).Order("created_at ASC").Offset(offset).Limit(limit).Find(&messages).Error
	return messages, err
}

// 私信列表
func (r *MessageRepository) GetConversations(userID uint) ([]model.Message, error) {
	var messages []model.Message
	query := `SELECT * FROM messages
            WHERE id IN (
                SELECT MAX(id) FROM messages
                               WHERE sender_id = ? OR receiver_id=?
                               GROUP BY session_id
            )
            ORDER BY created_at DESC 
            `
	err := r.DB.Raw(query, userID, userID).Scan(&messages).Error
	return messages, err
}

// 私信未读总数
func (r *MessageRepository) GetUnreadCountByUser(userID uint) (int64, error) {
	var count int64
	err := r.DB.Model(&model.Message{}).Where("receiver_id=? AND is_read = ?", userID, false).Count(&count).Error
	return count, err
}
func (r *MessageRepository) MarkMessagesAsRead(sessionID string, receiverID uint) error {
	return r.DB.Model(&model.Message{}).Where("session_id=? AND receiver_id = ? AND is_read =?", sessionID, receiverID, false).Update("is_read", true).Error
}
