package repository

import (
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
func (r *UserRepository) FindByID(id uint) (*model.User, error) {
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
func (r *PostRepository) FindByID(id uint) (*model.Post, error) {
	var post model.Post
	err := r.DB.Where("status IN ?", []int{0, 1}).Preload("Author").Preload("Comments.Author").First(&post, id).Error
	if err != nil {
		return nil, e.ErrInvalidArgs
	}
	if post.Status == 2 {
		return nil, e.ErrPostNotFound
	}
	return &post, err
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

// 获取指定用户草稿箱列表
func (r *PostRepository) ListDrafts(userID uint, offset, limit int) ([]model.Post, error) {
	var posts []model.Post
	//降序排列
	err := r.DB.Where("author_id = ? AND status=?", userID, 0).Order("updated_at DESC").Offset(offset).Limit(limit).Find(&posts).Error
	return posts, err
}

// 检查空字符，mysql的match语法不支持空字符
func cleanFullTextKeyword(keyword string) string {
	reg := regexp.MustCompile(`[+\-<>()~*@"\\]`)
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
	query := `SELECT id,title,created_at FROM posts WHERE MATCH(title,content) AGAINST(? IN BOOLEAN MODE) AND status=1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT ? OFFSET ?`
	err := r.DB.Raw(query, processedKeyword, limit, offset).Scan(&posts).Error
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
func (r *RelationRepository) IsFollowing(followerID, followeeID uint) (bool, error) {
	var count int64
	err := r.DB.Model(&model.Relation{}).Where("follower_id=? AND followee_id=?", followerID, followeeID).Count(&count).Error
	return count > 0, err
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

type FeedRepository struct {
	DB *gorm.DB
}

func NewFeedRepository(db *gorm.DB) *FeedRepository {
	return &FeedRepository{DB: db}
}

// 获取用户动态

func (r *FeedRepository) GetFeedByUserIDs(userIDs []uint, offset, limit int) ([]model.Post, error) {
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
	err := r.DB.Preload("Author").Order("hotscore DESC").Limit(limit).Find(&posts).Error
	return posts, err
}
