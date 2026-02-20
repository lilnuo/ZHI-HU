package repository

import (
	"context"
	"go-zhihu/internal/model"
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
func (r *UserRepository) CreateUser(ctx context.Context, tx *gorm.DB, user *model.User) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Create(user).Error
}
func (r *UserRepository) FindUsername(ctx context.Context, tx *gorm.DB, username string) (*model.User, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var user model.User
	err := db.WithContext(ctx).Where("username=?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
func (r *UserRepository) FindUserByID(ctx context.Context, tx *gorm.DB, id uint) (*model.User, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var user model.User
	err := db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// 个人信息更改
func (r *UserRepository) UpdateProfile(ctx context.Context, tx *gorm.DB, userID uint, avatar, bio string) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
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
	return db.WithContext(ctx).Model(&model.User{}).Where("id=?", userID).Updates(updates).Error
}

type PostRepository struct {
	DB *gorm.DB
}

func NewPostRepository(db *gorm.DB) *PostRepository {
	return &PostRepository{DB: db}
}

// 处理文章操作：发布、更新、获取详细、热度排序、搜
func (r *PostRepository) CreatePost(ctx context.Context, tx *gorm.DB, post *model.Post) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Create(post).Error
}
func (r *PostRepository) FindPostByID(ctx context.Context, tx *gorm.DB, id uint) (*model.Post, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var post model.Post
	err := db.WithContext(ctx).Where("status IN ?", []int{model.PostStatusDraft, model.PostStatusPublished}).Preload("Author").First(&post, id).Error
	if err != nil {
		return nil, err
	}

	return &post, err
}
func (r *PostRepository) FindPostsByIDs(ctx context.Context, tx *gorm.DB, ids []string) ([]model.Post, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var posts []model.Post
	err := db.WithContext(ctx).Where("id IN ?", ids).Where("status = ?", 1).Preload("Author").Find(&posts).Error
	return posts, err
}
func (r *PostRepository) UpdateStatus(ctx context.Context, tx *gorm.DB, postID uint, status int) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Model(&model.Post{}).Where("id = ?", postID).Update("status", status).Error
}
func (r *PostRepository) UpdatePost(ctx context.Context, tx *gorm.DB, post *model.Post) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Save(post).Error
}

func (r *PostRepository) ListPosts(ctx context.Context, tx *gorm.DB, offset, limit int, orderBy string) ([]model.Post, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var posts []model.Post
	err := db.WithContext(ctx).Where("status=?", model.PostStatusPublished).Preload("Author").Order(orderBy + " desc").Offset(offset).Limit(limit).Find(&posts).Error
	return posts, err
}

// 获取指定用户主页
func (r *PostRepository) ListPublicByAuthorID(ctx context.Context, tx *gorm.DB, authorID uint, offset, limit int) ([]model.Post, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var posts []model.Post
	err := db.WithContext(ctx).Where("author_id = ? AND status = ?", authorID, model.PostStatusPublished).Preload("Author").Order("created_at DESC").Offset(offset).Limit(limit).Find(&posts).Error
	return posts, err
}

// 获取指定用户草稿箱列表
func (r *PostRepository) ListDrafts(ctx context.Context, tx *gorm.DB, userID uint, offset, limit int) ([]model.Post, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var posts []model.Post
	//降序排列
	err := db.WithContext(ctx).Where("author_id = ? AND status=?", userID, model.PostStatusDraft).Order("updated_at DESC").Offset(offset).Limit(limit).Find(&posts).Error
	return posts, err
}

// 检查空字符，mysql的match语法不支持空字符
func cleanFullTextKeyword(keyword string) string {
	reg := regexp.MustCompile(`[^\p{Han}a-zA-Z0-9\s]`)
	return reg.ReplaceAllString(keyword, " ")
}
func (r *PostRepository) SearchPosts(ctx context.Context, tx *gorm.DB, keyword string, offset, limit int) ([]model.Post, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var posts []model.Post
	if strings.TrimSpace(keyword) == "" {
		return []model.Post{}, nil
	}
	processedKeyword := cleanFullTextKeyword(keyword)
	if strings.TrimSpace(processedKeyword) == "" {
		return []model.Post{}, nil
	}
	err := db.WithContext(ctx).Model(&model.Post{}).Preload("Author").Where("status=?", model.PostStatusPublished).Where("MATCH(title,content) AGAINST(? IN NATURAL LANGUAGE MODE)", keyword).Order("created_at DESC").Offset(offset).Limit(limit).Find(&posts).Error
	return posts, err
}

// 获取指定用户最近的文章id和创建时间
func (r *PostRepository) FindRecentPostIDsByAuthor(ctx context.Context, tx *gorm.DB, authorID uint, limit int) ([]model.Post, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var posts []model.Post
	err := db.WithContext(ctx).Model(&model.Post{}).Select("id,created_at").Where("author_id = ? AND status = ?", authorID, model.PostStatusPublished).Order("created_at DESC").Limit(limit).Find(&posts).Error
	return posts, err
}

type CommentRepository struct {
	DB *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{DB: db}
}

// 处理评论和回答的发布与对话
func (r *CommentRepository) CreateComment(ctx context.Context, tx *gorm.DB, comment *model.Comment) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Create(comment).Error
}
func (r *CommentRepository) GetCommentByPostID(ctx context.Context, tx *gorm.DB, postID uint) ([]model.Comment, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var comments []model.Comment
	err := db.WithContext(ctx).Where("post_id=?", postID).Preload("Author").Order("created_at asc").Find(&comments).Error
	return comments, err
}

type RelationRepository struct {
	DB *gorm.DB
}

func NewRelationRepository(db *gorm.DB) *RelationRepository {
	return &RelationRepository{DB: db}
}

// 处理关注关系
func (r *RelationRepository) Follow(ctx context.Context, tx *gorm.DB, followerID, followeeID uint) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
	relation := model.Relation{
		FollowerID: followerID,
		FolloweeID: followeeID,
	}
	return db.WithContext(ctx).Create(&relation).Error
}

func (r *RelationRepository) Unfollow(ctx context.Context, tx *gorm.DB, followerID, followeeID uint) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Where("follower_id=? AND followee_id=?", followerID, followeeID).Delete(&model.Relation{}).Error
}
func (r *RelationRepository) GetFolloweeIDs(ctx context.Context, tx *gorm.DB, userID uint) ([]uint, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var ids []uint
	err := db.WithContext(ctx).Model(&model.Relation{}).Where("follower_id=?", userID).Pluck("followee_id", &ids).Error
	return ids, err
}
func (r *RelationRepository) GetFollowerIDs(ctx context.Context, tx *gorm.DB, userID uint) ([]uint, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var ids []uint
	err := db.WithContext(ctx).Model(&model.Relation{}).Where("followee_id=?", userID).Pluck("follower_id", &ids).Error
	return ids, err
}
func (r *RelationRepository) IsFollowing(ctx context.Context, tx *gorm.DB, followerID, followeeID uint) (bool, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var count int64
	err := db.WithContext(ctx).Model(&model.Relation{}).Where("follower_id=? AND followee_id=?", followerID, followeeID).Count(&count).Error
	return count > 0, err
}

// 获取粉丝列表（谁关注了我）
func (r *RelationRepository) GetFollowers(ctx context.Context, tx *gorm.DB, userID uint, offset, limit int) ([]model.User, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var users []model.User
	err := db.WithContext(ctx).Table("users").Select("users.id,users.username,users.avatar,user.bio,users.created_at").Joins("JOIN relations ON user_id = relations.follower_id").Where("relations.followee_id=?", userID).Order("relations.created_at DESC").Offset(offset).Limit(limit).Find(&users).Error
	return users, err
}

// 获取关注列表
func (r *RelationRepository) GetFollowees(ctx context.Context, tx *gorm.DB, userID uint, offset, limit int) ([]model.User, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var users []model.User
	err := db.WithContext(ctx).Table("users").Select("users.id,users.username,users.avatar,users.created_at").Joins("JOIN relations ON users.id = relations.followee_id").Where("relations.follower_id =?", userID).Order("relations.created_at DESC").Offset(offset).Limit(limit).Find(&users).Error
	return users, err
}

type LikeRepository struct {
	DB *gorm.DB
}

func NewLikeRepository(db *gorm.DB) *LikeRepository {
	return &LikeRepository{DB: db}
}

// 处理点赞
func (r *LikeRepository) AddLike(like *model.Like, ctx context.Context, tx *gorm.DB) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Create(like).Error
}
func (r *LikeRepository) RemoveLike(ctx context.Context, tx *gorm.DB, userID, targetID uint, likeType int) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Where("user_id=? AND target_id=? AND type=?", userID, targetID, likeType).Delete(&model.Like{}).Error
}
func (r *LikeRepository) CountLikes(ctx context.Context, tx *gorm.DB, targetID uint) (int64, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var count int64
	err := db.WithContext(ctx).Model(&model.Like{}).Where("target_id=? AND type=1", targetID).Count(&count).Error
	return count, err
}
func (r *LikeRepository) IsLike(ctx context.Context, tx *gorm.DB, userID, targetID uint, targetType int) (bool, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var count int64
	err := db.WithContext(ctx).Model(&model.Like{}).Where("user_id=? AND target_id=? AND type=?", userID, targetID, targetType).Count(&count).Error
	return count > 0, err
}

func (r *PostRepository) UpdateHotScore(ctx context.Context, tx *gorm.DB, postID uint, scoreDelta float64) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Model(&model.Post{}).Where("id=?", postID).Update("hot_score", gorm.Expr("hot_score + ?", scoreDelta)).Error
}
func (r *LikeRepository) FindLike(ctx context.Context, userID, targetID uint, likeType int, tx *gorm.DB) (*model.Like, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var like model.Like
	err := db.WithContext(ctx).Where("user_id = ? AND target_id = ? AND type = ?", userID, targetID, likeType).First(&like).Error
	if err != nil {
		return nil, err
	}
	return &like, nil
}

// 事务方法
//func (r *LikeRepository) ToggleLikeWithTx(ctx context.Context,tx *gorm.DB,userID, targetID uint, targetType int) (bool, bool, error) {
//	db:=r.DB
//	if tx!=nil{
//		db=tx
//	}
//	var isLike bool
//	var isNewAction bool
//	err := r.DB.Transaction(func(tx *gorm.DB) error {
//		var existingLike model.Like
//		err := tx.Where("user_id = ? AND target_id = ? AND type =?", userID, targetID, targetType).First(existingLike).Error
//
//		const likePostScore = 10
//		const likeCommentScore = 5
//		if err == nil {
//			isNewAction = false
//			isLike = false
//			if err := tx.Delete(&existingLike).Error; err != nil {
//				return err
//			}
//			if targetType == 1 {
//				if err := tx.Model(&model.Post{}).Where("id = ?", targetID).Update("hot_score", gorm.Expr("hot_score - ?", likePostScore)).Error; err != nil {
//					return err
//				}
//			} else if targetType == 2 {
//				var comment model.Comment
//				if err := tx.Select("post_id").Where("id = ?", targetID).First(&comment).Error; err != nil {
//					return err
//				}
//			}
//			return nil
//		} else if errors.Is(err, gorm.ErrRecordNotFound) {
//			isNewAction = true
//			isLike = true
//			like := &model.Like{UserID: userID, TargetID: targetID, Type: targetType}
//			if err := tx.Create(like).Error; err != nil {
//				return err
//			}
//			if targetType == 1 {
//				if err := tx.Model(&model.Post{}).Where("id = ?", targetID).Update("hot_score + ?", likePostScore).Error; err != nil {
//					return err
//				}
//			} else if targetType == 2 {
//				var comment model.Comment
//				if err := tx.Select("post_id").Where("id = ?", targetID).First(&comment).Error; err != nil {
//					return nil
//				}
//				if err := tx.Model(&model.Post{}).Where("id = ?", comment.PostID).Update("hot_score", gorm.Expr("hot_score+ ?", likeCommentScore)).Error; err != nil {
//					return nil
//				}
//			}
//			return nil
//		}
//		return err
//
//	})
//	return isLike, isNewAction, err
//}太复杂了吧

type FeedRepository struct {
	DB *gorm.DB
}

func NewFeedRepository(db *gorm.DB) *FeedRepository {
	return &FeedRepository{DB: db}
}

// 获取用户动态

func (r *FeedRepository) GetFeedByUserIDs(ctx context.Context, tx *gorm.DB, userIDs []uint, offset, limit int) ([]model.Post, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	if len(userIDs) == 0 {
		return []model.Post{}, nil
	}
	var posts []model.Post
	err := db.WithContext(ctx).Where("author_id IN ?", userIDs).Where("status=1").Preload("Author").Order("created_at DESC").Offset(offset).Limit(limit).Find(&posts).Error
	return posts, err
}

// 禁言处理补充
func (r *UserRepository) BanUser(ctx context.Context, tx *gorm.DB, id uint) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Model(&model.User{}).Where("id=?", id).Update("status", 0).Error
}
func (r *UserRepository) UnbanUser(ctx context.Context, tx *gorm.DB, id uint) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Model(&model.User{}).Where("id=?", id).Update("status", 1).Error
}
func (r *UserRepository) IsUserBanned(ctx context.Context, tx *gorm.DB, id uint) (bool, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var user model.User
	err := db.WithContext(ctx).Select("status").First(&user, id).Error
	if err != nil {
		return false, err
	}
	return user.Status == 0, nil
}

// 排行榜补充
func (r *PostRepository) GetLeaderboard(ctx context.Context, tx *gorm.DB, limit int) ([]model.Post, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var posts []model.Post
	err := db.WithContext(ctx).Where("status=?", 1).Preload("Author").Order("hot_score DESC").Limit(limit).Find(&posts).Error
	return posts, err
}

type ConnectRepository struct {
	DB *gorm.DB
}

func NewConnectionRepository(db *gorm.DB) *ConnectRepository {
	return &ConnectRepository{DB: db}
}

// 关注文章或问题
func (r ConnectRepository) AddConnection(ctx context.Context, tx *gorm.DB, userID, postID uint) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
	conn := &model.Connection{
		UserID: userID,
		PostID: postID,
	}
	return db.WithContext(ctx).Create(conn).Error
}
func (r *ConnectRepository) RemoveConn(ctx context.Context, tx *gorm.DB, userID, postID uint) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Where("user_id=? AND post_id =?", userID, postID).Delete(&model.Connection{}).Error
}

func (r *ConnectRepository) IsConn(ctx context.Context, tx *gorm.DB, userID, postID uint) (bool, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var count int64
	err := db.WithContext(ctx).Model(&model.Connection{}).Where("user_id=? AND post_id=?", userID, postID).Count(&count).Error
	return count > 0, err
}

// 获取关注收藏列表
func (r *ConnectRepository) GetConnByUser(ctx context.Context, tx *gorm.DB, userID uint, offset, limit int) ([]model.Post, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var posts []model.Post
	err := db.WithContext(ctx).Table("posts").Joins("JOIN connections ON posts.id =connections.post_id").Where("connections.user_id=?", userID).Order("connections.created_at DESC").Offset(offset).Limit(limit).Find(&posts).Error
	return posts, err
}
func (r *CommentRepository) FindCommentByID(ctx context.Context, tx *gorm.DB, id uint) (*model.Comment, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var comment model.Comment
	err := db.WithContext(ctx).First(&comment, id).Error
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
func (r *NotificationRepository) CreateNotification(ctx context.Context, tx *gorm.DB, n *model.Notification) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Create(n).Error
}
func (r *NotificationRepository) GetNotifications(ctx context.Context, tx *gorm.DB, userID uint, offset, limit int) ([]model.Notification, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var notifications []model.Notification
	err := db.WithContext(ctx).Where("recipient_id =?", userID).Preload("Actor").Order("created_at DESC").Offset(offset).Limit(limit).Find(&notifications).Error
	return notifications, err
}

// 红标信息
func (r *NotificationRepository) GetUnreadCount(ctx context.Context, tx *gorm.DB, userID uint) (int64, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var count int64
	err := db.WithContext(ctx).Model(&model.Notification{}).Where("recipient_id = ? AND is_read=?", userID, false).Count(&count).Error
	return count, err
}
func (r *NotificationRepository) MarkAsRead(ctx context.Context, tx *gorm.DB, notificationID, userID uint) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Model(&model.Notification{}).Where("id =? AND recipient_id =?", notificationID, userID).Update("is_read", true).Error
}
func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, tx *gorm.DB, userID uint) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Model(&model.Notification{}).Where("recipient_id =? AND is_read= ?", userID, false).Update("is_read", true).Error
}
func (r *NotificationRepository) DeleteNotification(ctx context.Context, tx *gorm.DB, id uint) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Delete(&model.Notification{}, id).Error
}

// 关注私信
type MessageRepository struct {
	DB *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{DB: db}
}
func (r *MessageRepository) CreateMessage(ctx context.Context, tx *gorm.DB, msg *model.Message) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Create(msg).Error
}
func (r *MessageRepository) GetMessageBySession(ctx context.Context, tx *gorm.DB, sessionID string, offset, limit int) ([]model.Message, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var messages []model.Message
	err := db.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at ASC").Offset(offset).Limit(limit).Find(&messages).Error
	return messages, err
}

// 私信列表
func (r *MessageRepository) GetConversations(ctx context.Context, tx *gorm.DB, userID uint) ([]model.Message, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var messages []model.Message
	query := `SELECT * FROM messages
            WHERE id IN (
                SELECT MAX(id) FROM messages
                               WHERE sender_id = ? OR receiver_id=?
                               GROUP BY session_id
            )
            ORDER BY created_at DESC 
            `
	err := db.WithContext(ctx).Raw(query, userID, userID).Scan(&messages).Error
	return messages, err
}

// 私信未读总数
func (r *MessageRepository) GetUnreadCountByUser(ctx context.Context, tx *gorm.DB, userID uint) (int64, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var count int64
	err := db.WithContext(ctx).Model(&model.Message{}).Where("receiver_id=? AND is_read = ?", userID, false).Count(&count).Error
	return count, err
}
func (r *MessageRepository) MarkMessagesAsRead(ctx context.Context, tx *gorm.DB, sessionID string, receiverID uint) error {
	db := r.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Model(&model.Message{}).Where("session_id=? AND receiver_id = ? AND is_read =?", sessionID, receiverID, false).Update("is_read", true).Error
}
