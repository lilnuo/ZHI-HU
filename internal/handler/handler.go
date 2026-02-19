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
// Register 用户注册
// @Summary 用户注册
// Description 创建新用户账号
// @Tags 用户认证
// @Accept json
// @Produce json
// @Param data body  RegisterReq true "注册信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /register [post]# 替换 username 为你的 Docker Hub 用户名
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

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录获取Token
// @Tags 用户认证
// @Accept json
// @Produce json
// @Param data body LoginReq true "登录信息"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /login [post]
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
// UpdateProfile 更新个人信息
// @Summary 更新个人信息
// @Description 更新当前用户的头像、简介或状态
// @Tags 用户
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param data body UpdateProfileRe true "更新信息"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /user/profile [put]
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

// CreatPost 创建文章
// @Summary 创建文章或问题
// @Description 用户发布新的内容
// @Tags 文章
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param data body CreatePostRequest true "文章内容"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /user/posts [post]
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
// GetDrafts 获取草稿箱列表
// @Summary 获取草稿箱列表
// @Description 获取当前用户的草稿列表
// @Tags 文章
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /user/posts/drafts [get]
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
// GetLatestPosts 获取最新文章列表
// @Summary 获取最新文章列表
// @Description 获取最新发布的文章或问题列表
// @Tags 文章
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /user/posts/lists [get]
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
// GetComments 获取评论列表
// @Summary 获取文章评论
// @Description 根据文章ID获取评论列表
// @Tags 互动
// @Accept json
// @Produce json
// @Param post_id path int true "文章ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Router /user/posts/{post_id} [get]
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

// GetPostDetail 获取文章详情
// @Summary 获取文章详情
// @Description 根据ID获取文章详情
// @Tags 文章
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /posts/{id} [get]
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

// UpdatePost 更新文章
// @Summary 更新文章
// @Description 更新指定ID的文章内容
// @Tags 文章
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "文章ID"
// @Param data body CreatePostRequest true "文章内容"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /user/posts/{id} [put]
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
// PublishPost 发布草稿
// @Summary 发布草稿
// @Description 将草稿状态的变更为发布状态
// @Tags 文章
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "文章ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /user/posts/{id}/publish [post]
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

// DeletePost 删除文章
// @Summary 删除文章
// @Description 删除指定ID的文章
// @Tags 文章
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "文章ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /user/posts/{id} [delete]
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

// Search 搜索文章
// @Summary 搜索文章
// @Description 根据关键词搜索文章列表
// @Tags 文章
// @Accept json
// @Produce json
// @Param keyword query string false "搜索关键词"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /posts/search [get]
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

// FollowUser 关注用户
// @Summary 关注用户
// @Description 关注指定ID的用户
// @Tags 用户关系
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "被关注用户ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /user/follow/{id} [post]
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

// UnFollowUser 取消关注用户
// @Summary 取消关注用户
// @Description 取消关注指定ID的用户
// @Tags 用户关系
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "被取消关注用户ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /user/unfollow/{id}/ [post]
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

// AddComment 添加评论
// @Summary 添加评论
// @Description 对指定文章添加评论
// @Tags 互动
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "文章ID"
// @Param data body AddCommentRequest true "评论内容"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /user/posts/{id}/comments [post]
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

// GetFeed 获取Feed流
// @Summary 获取关注动态
// @Description 获取当前用户关注的人的动态流
// @Tags Feed
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /user/feed [get]
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

// BanUser 封禁用户
// @Summary 封禁用户
// @Description 管理员封禁指定用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "用户ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /admin/ban/{id} [post]
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
// UnbanUser 解禁用户
// @Summary 解禁用户
// @Description 管理员解禁指定用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "用户ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /admin/ban/{id} [post]
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
// GetLeaderboard 获取排行榜
// @Summary 获取排行榜
// @Description 获取热门文章排行榜
// @Tags 文章
// @Accept json
// @Produce json
// @Param limit query int false "数量限制" default(10)
// @Success 200 {object} map[string]interface{} "成功"
// @Router /posts/ranking [get]
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
// GetFollowers 获取粉丝列表
// @Summary 获取粉丝列表
// @Description 获取当前用户的粉丝列表
// @Tags 用户关系
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /user/followers [get]
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

// GetFollowees 获取关注列表
// @Summary 获取关注列表
// @Description 获取当前用户关注的用户列表
// @Tags 用户关系
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /user/following [get]
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
// ToggleConn 收藏/取消收藏文章
// @Summary 收藏/取消收藏文章
// @Description 切换对文章的收藏状态
// @Tags 互动
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "文章ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /user/connection/{id} [post]
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

// GetConn 获取收藏列表
// @Summary 获取收藏列表
// @Description 获取当前用户的收藏文章列表
// @Tags 互动
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /user/collections [get]
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

// ToggleLike 点赞/取消点赞
// @Summary 点赞/取消点赞
// @Description 对文章或评论进行点赞/取消点赞操作
// @Tags 互动
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param data body ToggleLikeRequest true "点赞信息"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /user/like [post]
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
// GetNotifications 获取通知列表
// @Summary 获取通知列表
// @Description 获取当前用户的系统通知列表
// @Tags 通知
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /user/notifications [get]
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
// GetUnreadCount 获取未读通知数量
// @Summary 获取未读通知数量
// @Description 获取当前用户的未读系统通知数量
// @Tags 通知
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /user/notifications/unread [get]
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
// MarkNotificationRead 标记通知为已读
// @Summary 标记通知为已读
// @Description 将指定ID的通知标记为已读
// @Tags 通知
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "通知ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /user/notifications/read/{id} [put]
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

// MarkAllRead 全部标记已读
// @Summary 全部标记已读
// @Description 将当前用户的所有通知标记为已读
// @Tags 通知
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /user/notifications/read/read-all [put]
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

// SendMsg 发送私信
// @Summary 发送私信
// @Description 给指定用户发送私信
// @Tags 消息
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param data body SendMsgRequest true "消息内容"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /user/messages [post]
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
// GetChatHistory 获取聊天记录
// @Summary 获取聊天记录
// @Description 获取与指定用户的聊天记录
// @Tags 消息
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "对方用户ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /user/messages/{id} [get]
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
// GetConversations 获取会话列表
// @Summary 获取会话列表
// @Description 获取当前用户的私信会话列表
// @Tags 消息
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /user/messsages/conversations [get]
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

// GetTotalUnread 获取私信未读总数
// @Summary 获取私信未读总数
// @Description 获取当前用户的私信未读消息总数
// @Tags 消息
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /user/messages/unread [get]
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

// 获取指定用户基本资料以及公开文章
// GetUserProfile 获取指定用户资料
// @Summary 获取指定用户资料
// @Description 获取指定ID用户的公开资料
// @Tags 用户
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Router /users/{id}/profile [get]
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

// GetUserPosts 获取指定用户文章
// @Summary 获取指定用户文章
// @Description 获取指定ID用户发布的公开文章列表
// @Tags 用户
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /users/{id}/posts [get]
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
