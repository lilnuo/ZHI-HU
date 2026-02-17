package handler

import (
	"go-zhihu/internal/service"
	"go-zhihu/pkg/e"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	Service *service.Service
	db      *gorm.DB
}

func NewHandler(Service *service.Service, db *gorm.DB) *Handler {
	return &Handler{Service: Service, db: db}
}

// Register 用户注册
// @Summary 用户注册
// Description 创建新用户账号
// @Tags 用户认证
// @Accept json
// @Produce json
// @Param data body  RegisterReq true "注册信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Route /register [post]
type RegisterReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
	Email    string `json:"email" binding:"required,email"`
}
type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}
type UpdateProfileRe struct {
	Avatar string `json:"avatar" binding:"omitempty"`
	Bio    string `json:"bio" binding:"omitempty,max=500"`
	Status *int   `json:"status"` //用*区分不传与传0
}

// 获取id并验证函数，减少重复代码
func getUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		e.ErrorResponse(c, e.ErrUnAuthorizedInstance)
		return 0, false
	}
	uid, ok := userID.(uint)
	if !ok {
		e.ErrorResponse(c, e.ErrServer)
		return 0, false
	}
	return uid, true
}
func parseIDParam(c *gin.Context, paramKey string) (uint, error) {
	paramID := c.Param(paramKey)
	id, err := strconv.ParseUint(paramID, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

// 注册与登陆
func (h *Handler) Register(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.Service.User.Register(ctx, tx, req.Username, req.Password, req.Email); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, e.ErrSuccess)
}
func (h *Handler) Login(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	resp, err := h.Service.User.Login(ctx, tx, req.Username, req.Password)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, resp)
}

// 更新个人信息
func (h *Handler) UpdateProfile(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	var req UpdateProfileRe
	if err := c.ShouldBindJSON(&req); err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.Service.User.UpdateProfile(ctx, tx, uid, req.Avatar, req.Bio); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}

// 处理文章/问题·相关
type CreatePostRequest struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
	Type    int    `json:"type" binding:"required,oneof=1 2"` //1.chapter 2.question
	Status  int    `json:"status" binding:"required"`
}

func (h *Handler) CreatPost(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.Service.Post.CreatePost(ctx, tx, uid, req.Title, req.Content, req.Type, req.Status); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}

// 获取草稿箱列表
func (h *Handler) GetDrafts(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if page < 1 {
		page = 1
	}
	if pageSize > 50 {
		pageSize = 50
	}
	drafts, err := h.Service.Post.GetDrafts(ctx, tx, uid, page, pageSize)
	if err != nil {
		e.ErrorResponse(c, e.ErrServer)
		return
	}
	e.SuccessResponse(c, drafts)
}

// 获取最新文章列表
func (h *Handler) GetLatestPosts(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	posts, err := h.Service.Post.GetLatestPosts(ctx, tx, page, pageSize)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, posts)
}

// 看评论
func (h *Handler) GetComments(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	postID, err := parseIDParam(c, "post_id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	comments, err := h.Service.Interaction.GetComments(ctx, tx, postID)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, comments)
}

func (h *Handler) GetPostDetail(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	postID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	post, err := h.Service.Post.GetPostDetail(ctx, tx, postID)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, post)
}
func (h *Handler) UpdatePost(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	postID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
	}
	if err := h.Service.Post.UpdatePost(ctx, tx, postID, uid, req.Title, req.Content, &req.Status); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}

// 发布草稿
func (h *Handler) PublishPost(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	postID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.Service.Post.PublishPost(ctx, tx, postID, uid); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}
func (h *Handler) DeletePost(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	postID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.Service.Post.DeletePost(ctx, tx, postID, uid); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}
func (h *Handler) Search(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	keyword := c.DefaultQuery("keyword", "")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	posts, err := h.Service.Post.Search(ctx, tx, keyword, page, pageSize)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, posts)
}
func (h *Handler) FollowUser(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	targetID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
	}
	if err := h.Service.Relation.FollowUser(ctx, tx, uid, targetID); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)

}
func (h *Handler) UnFollowUser(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	targetID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.Service.Relation.UnfollowUser(ctx, tx, uid, targetID); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}

type AddCommentRequest struct {
	Content string `json:"content" binding:"required"`
}

func (h *Handler) AddComment(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	postID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	var req AddCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.Service.Interaction.AddComment(ctx, tx, postID, uid, req.Content); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}
func (h *Handler) GetFeed(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	posts, err := h.Service.Feed.GetFeed(ctx, tx, uid, page, pageSize)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, posts)
}
func (h *Handler) BanUser(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	_, ok := getUserID(c)
	if !ok {
		return
	}
	targetID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.Service.User.BanUser(ctx, tx, targetID); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}

// 解禁补充
func (h *Handler) UnbanUser(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	_, ok := getUserID(c)
	if !ok {
		return
	}
	targetID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.Service.User.UnbanUser(ctx, tx, targetID); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}

// 排行榜补充
func (h *Handler) GetLeaderboard(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	posts, err := h.Service.Post.GetLeaderboard(ctx, tx, limit)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, posts)
}

// 获取粉丝或关注列表
func (h *Handler) GetFollowers(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if page < 1 {
		page = 1
	}
	if pageSize > 100 {
		pageSize = 100
	}
	users, err := h.Service.Relation.GetFollowers(ctx, tx, uid, page, pageSize)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, users)
}
func (h *Handler) GetFollowees(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if page < 1 {
		page = 1
	}
	if pageSize > 100 {
		pageSize = 100
	}
	users, err := h.Service.Relation.GetFollowees(ctx, tx, uid, page, pageSize)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, users)
}

// 关注收藏文章或问题
func (h *Handler) ToggleConn(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	postID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.Service.Interaction.ToggleConn(ctx, tx, uid, postID); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}
func (h *Handler) GetConn(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if page < 1 {
		page = 1
	}
	if pageSize > 50 {
		pageSize = 50
	}
	posts, err := h.Service.Interaction.GetConn(ctx, tx, uid, page, pageSize)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, posts)
}

// 点赞请求结构体，分清楚类型
type ToggleLikeRequest struct {
	TargetID uint `json:"target_id" binding:"required"`
	Type     int  `json:"type" binding:"required,oneof=1 2"` //1:文章/问题·,2:评论
}

func (h *Handler) ToggleLike(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	var req ToggleLikeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.Service.Interaction.ToggleLike(ctx, tx, uid, req.TargetID, req.Type); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}

// 获取通知列表
func (h *Handler) GetNotifications(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	list, err := h.Service.Notification.GetNotifications(ctx, tx, uid, page, pageSize)
	if err != nil {
		e.ErrorResponse(c, e.ErrServer)
		return
	}
	e.SuccessResponse(c, list)
}

// 红标数量
func (h *Handler) GetUnreadCount(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	count, err := h.Service.Notification.GetUnreadCount(ctx, tx, uid)
	if err != nil {
		e.ErrorResponse(c, e.ErrServer)
		return
	}
	e.SuccessResponse(c, gin.H{"unread_count": count})
}

// 单条已读
func (h *Handler) MarkNotificationRead(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	id, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.Service.Notification.MarkNotificationRead(ctx, tx, id, uid); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}
func (h *Handler) MarkAllRead(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	if err := h.Service.Notification.MarkAllNotificationsRead(ctx, tx, uid); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}

// 私信通知
type SendMsgRequest struct {
	ReceiverID uint   `json:"receiver_id" binding:"required"`
	Content    string `json:"content" binding:"required,min=1"`
}

func (h *Handler) SendMsg(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	var req SendMsgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.Service.Message.SendMessage(ctx, tx, uid, req.ReceiverID, req.Content); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
} //聊天记录
func (h *Handler) GetChatHistory(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	peerIDStr := c.Param("id")
	peerID, err := strconv.ParseUint(peerIDStr, 10, 32)
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	} //换个样子
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	history, err := h.Service.Message.GetChatHistory(ctx, tx, uid, uint(peerID), page, pageSize)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, history)
}

// 会话列表
func (h *Handler) GetConversations(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	list, err := h.Service.Message.GetConversations(ctx, tx, uid)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, list)
}
func (h *Handler) GetTotalUnread(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	counts, err := h.Service.Message.GetTotalUnread(ctx, tx, uid)
	if err != nil {
		e.ErrorResponse(c, e.ErrServer)
		return
	}
	e.SuccessResponse(c, counts)
}

//获取指定用户基本资料以及公开文章

func (h *Handler) GetUserProfile(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	targetID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	profile, err := h.Service.User.GetUserProfile(ctx, tx, targetID)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, profile)
}
func (h *Handler) GetUserPosts(c *gin.Context) {
	ctx := c.Request.Context()
	tx := h.db
	targetID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize > 50 {
		pageSize = 50
	}
	posts, err := h.Service.Post.GetUserPosts(ctx, tx, targetID, page, pageSize)
	if err != nil {
		e.ErrorResponse(c, e.ErrServer)
		return
	}
	e.SuccessResponse(c, posts)

}
