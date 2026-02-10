package handler

import (
	"go-zhihu/internal/service"
	"go-zhihu/pkg/e"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	AuthService *service.SocialService
}

func NewUserHandler(authService *service.SocialService) *Handler {
	return &Handler{AuthService: authService}
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
func (h *Handler) Register(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.AuthService.Register(req.Username, req.Password, req.Email); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, e.ErrSuccess)
}
func (h *Handler) Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	resp, err := h.AuthService.Login(req.Username, req.Password)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, resp)
}

// 更新个人信息
func (h *Handler) UpdateProfile(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	var req UpdateProfileRe
	if err := c.ShouldBindJSON(&req); err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.AuthService.UpdateProfile(uid, req.Avatar, req.Bio); err != nil {
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
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.AuthService.CreatePost(uid, req.Title, req.Content, req.Type, req.Status); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}

// 获取草稿箱列表
func (h *Handler) GetDrafts(c *gin.Context) {
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
	drafts, err := h.AuthService.GetDrafts(uid, page, pageSize)
	if err != nil {
		e.ErrorResponse(c, e.ErrServer)
		return
	}
	e.SuccessResponse(c, drafts)
}

// 获取最新文章列表
func (h *Handler) GetLatestPosts(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	posts, err := h.AuthService.GetLatestPosts(page, pageSize)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, posts)
}

// 看评论
func (h *Handler) GetComments(c *gin.Context) {
	postID, err := parseIDParam(c, "post_id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	comments, err := h.AuthService.GetComments(postID)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, comments)
}

func (h *Handler) GetPostDetail(c *gin.Context) {
	postID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	post, err := h.AuthService.GetPostDetail(postID)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, post)
}
func (h *Handler) UpdatePost(c *gin.Context) {
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
	if err := h.AuthService.UpdatePost(postID, uid, req.Title, req.Content, &req.Status); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}

// 发布草稿
func (h *Handler) PublishPost(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	postID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.AuthService.PublishPost(postID, uid); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}
func (h *Handler) DeletePost(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	postID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.AuthService.DeletePost(postID, uid); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}
func (h *Handler) Search(c *gin.Context) {
	keyword := c.DefaultQuery("keyword", "")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	posts, err := h.AuthService.Search(keyword, page, pageSize)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, posts)
}
func (h *Handler) FollowUser(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	targetID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
	}
	if err := h.AuthService.FollowUser(uid, targetID); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)

}
func (h *Handler) UnFollowUser(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	targetID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.AuthService.UnfollowUser(uid, targetID); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}

func (h *Handler) ToggleLike(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	targetID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	const likeType = 1
	if err := h.AuthService.ToggleLike(uid, targetID, likeType); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}

type AddCommentRequest struct {
	Content string `json:"content" binding:"required"`
}

func (h *Handler) AddComment(c *gin.Context) {
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
	if err := h.AuthService.AddComment(postID, uid, req.Content); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}
func (h *Handler) GetFeed(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	posts, err := h.AuthService.GetFeed(uid, page, pageSize)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, posts)
}
func (h *Handler) BanUser(c *gin.Context) {
	_, ok := getUserID(c)
	if !ok {
		return
	}
	targetID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.AuthService.BanUser(targetID); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}

// 解禁补充
func (h *Handler) UnbanUser(c *gin.Context) {
	_, ok := getUserID(c)
	if !ok {
		return
	}
	targetID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.AuthService.UnbanUser(targetID); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}

// 排行榜补充
func (h *Handler) GetLeaderboard(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	posts, err := h.AuthService.GetLeaderboard(limit)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, posts)
}

// 获取粉丝或关注列表
func (h *Handler) GetFollowers(c *gin.Context) {
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
	users, err := h.AuthService.GetFollowers(uid, page, pageSize)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, users)
}
func (h *Handler) GetFollowees(c *gin.Context) {
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
	users, err := h.AuthService.GetFollowees(uid, page, pageSize)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, users)
}

// 关注收藏文章或问题
func (h *Handler) ToggleConn(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	postID, err := parseIDParam(c, "id")
	if err != nil {
		e.ErrorResponse(c, e.ErrInvalidArgs)
		return
	}
	if err := h.AuthService.ToggleConn(uid, postID); err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, nil)
}
func (h *Handler) GetConn(c *gin.Context) {
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
	posts, err := h.AuthService.GetConn(uid, page, pageSize)
	if err != nil {
		e.ErrorResponse(c, err)
		return
	}
	e.SuccessResponse(c, posts)
}
