package handlers

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"websitego/internal/models"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type DashboardHandler struct {
	db *gorm.DB
	mu sync.RWMutex

	cachedTotalUsers int64
	totalUsersLoaded bool
	searchAnchors    map[string]map[int]uint
}

const defaultPerPage = 10

var allowedPerPage = []int{10, 25, 50, 100}
var allowedSearchBy = map[string]struct{}{
	"all":   {},
	"id":    {},
	"name":  {},
	"email": {},
	"role":  {},
}

type UserListItem struct {
	ID    uint
	Name  string
	Email string
	Role  string
}

func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{
		db:            db,
		searchAnchors: make(map[string]map[int]uint),
	}
}

// WarmupTotalUsers preloads total user count so first dashboard request is fast.
func (h *DashboardHandler) WarmupTotalUsers() error {
	_, err := h.getTotalUsers()
	return err
}

func (h *DashboardHandler) Index(c *gin.Context) {
	h.renderDashboard(c, "")
}

func (h *DashboardHandler) CreateUser(c *gin.Context) {
	page, perPage := getPagination(c)
	searchQ := getSearchQuery(c)
	searchBy := getSearchBy(c)

	if !isAdmin(c) {
		c.Redirect(http.StatusFound, buildDashboardURL(page, perPage, searchQ, searchBy, "error", "Hanya admin yang bisa menambah user"))
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
	password := c.PostForm("password")
	role := normalizeRole(c.PostForm("role"))
	if name == "" || email == "" || len(password) < 6 {
		h.renderDashboard(c, "Nama, email, dan password minimal 6 karakter wajib diisi.")
		return
	}

	var existing models.User
	if err := h.db.Where("email = ?", email).First(&existing).Error; err == nil {
		h.renderDashboard(c, "Email sudah terdaftar.")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		h.renderDashboard(c, "Gagal memproses password.")
		return
	}

	user := models.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		Role:         role,
	}
	if err := h.db.Create(&user).Error; err != nil {
		h.renderDashboard(c, "Gagal menambah user.")
		return
	}
	h.adjustTotalUsersCache(1)

	c.Redirect(http.StatusFound, buildDashboardURL(page, perPage, searchQ, searchBy, "success", "User berhasil ditambahkan"))
}

func (h *DashboardHandler) UpdateUser(c *gin.Context) {
	page, perPage := getPagination(c)
	searchQ := getSearchQuery(c)
	searchBy := getSearchBy(c)

	if !isAdmin(c) {
		c.Redirect(http.StatusFound, buildDashboardURL(page, perPage, searchQ, searchBy, "error", "Hanya admin yang bisa mengubah user"))
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		h.renderDashboard(c, "ID user tidak valid.")
		return
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		h.renderDashboard(c, "User tidak ditemukan.")
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
	role := normalizeRole(c.PostForm("role"))
	password := c.PostForm("password")

	if name == "" || email == "" {
		h.renderDashboard(c, "Nama dan email wajib diisi.")
		return
	}

	var duplicate models.User
	if err := h.db.Where("email = ? AND id <> ?", email, user.ID).First(&duplicate).Error; err == nil {
		h.renderDashboard(c, "Email sudah digunakan user lain.")
		return
	}

	user.Name = name
	user.Email = email
	user.Role = role
	if strings.TrimSpace(password) != "" {
		if len(password) < 6 {
			h.renderDashboard(c, "Password baru minimal 6 karakter.")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			h.renderDashboard(c, "Gagal memproses password baru.")
			return
		}
		user.PasswordHash = string(hash)
	}

	if err := h.db.Save(&user).Error; err != nil {
		h.renderDashboard(c, "Gagal mengubah user.")
		return
	}

	currentUserID := sessionUserID(c)
	if currentUserID == user.ID {
		session := sessions.Default(c)
		session.Set("user_name", user.Name)
		session.Set("user_role", user.Role)
		_ = session.Save()
	}

	c.Redirect(http.StatusFound, buildDashboardURL(page, perPage, searchQ, searchBy, "success", "User berhasil diubah"))
}

func (h *DashboardHandler) DeleteUser(c *gin.Context) {
	page, perPage := getPagination(c)
	searchQ := getSearchQuery(c)
	searchBy := getSearchBy(c)

	if !isAdmin(c) {
		c.Redirect(http.StatusFound, buildDashboardURL(page, perPage, searchQ, searchBy, "error", "Hanya admin yang bisa menghapus user"))
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		h.renderDashboard(c, "ID user tidak valid.")
		return
	}

	currentUserID := sessionUserID(c)
	if currentUserID == uint(id) {
		h.renderDashboard(c, "Admin tidak bisa menghapus akun sendiri.")
		return
	}

	if err := h.db.Delete(&models.User{}, id).Error; err != nil {
		h.renderDashboard(c, "Gagal menghapus user.")
		return
	}
	h.adjustTotalUsersCache(-1)

	c.Redirect(http.StatusFound, buildDashboardURL(page, perPage, searchQ, searchBy, "success", "User berhasil dihapus"))
}

func (h *DashboardHandler) renderDashboard(c *gin.Context, formError string) {
	h.renderDashboardWithSuccess(c, formError, "")
}

func (h *DashboardHandler) renderDashboardWithSuccess(c *gin.Context, formError, formSuccess string) {
	page, perPage := getPagination(c)
	searchQ := getSearchQuery(c)
	searchBy := getSearchBy(c)
	searchMode := strings.TrimSpace(searchQ) != ""

	session := sessions.Default(c)
	name, _ := session.Get("user_name").(string)
	if strings.TrimSpace(name) == "" {
		name = "User"
	}
	role, _ := session.Get("user_role").(string)
	if strings.TrimSpace(role) == "" {
		role = "user"
	}

	if page < 1 {
		page = 1
	}

	totalUsers := int64(0)
	totalPages := 1
	if !searchMode {
		var err error
		totalUsers, err = h.countUsers(searchQ, searchBy)
		if err != nil {
			c.String(http.StatusInternalServerError, "gagal menghitung data users")
			return
		}
		totalPages = int((totalUsers + int64(perPage) - 1) / int64(perPage))
		if totalPages < 1 {
			totalPages = 1
		}
		if page > totalPages {
			page = totalPages
		}
	}

	offset := (page - 1) * perPage
	cacheKey := h.searchAnchorKey(searchQ, searchBy, perPage)

	listQuery := applySearchFilter(h.db.Model(&models.User{}), searchQ, searchBy)
	if searchMode && page > 1 {
		if anchorPage, anchorID, ok := h.getNearestSearchAnchor(cacheKey, page); ok && anchorID > 0 {
			listQuery = listQuery.Where("id >= ?", anchorID)
			offset = (page - anchorPage) * perPage
		}
	}
	var users []UserListItem
	limit := perPage
	if searchMode {
		// Probe one extra row to know whether "Next" should stay active.
		limit = perPage + 1
	}
	if err := listQuery.
		Select("id, name, email, role").
		Order("id ASC").
		Limit(limit).
		Offset(offset).
		Scan(&users).Error; err != nil {
		c.String(http.StatusInternalServerError, "gagal memuat data users")
		return
	}
	if searchMode && len(users) > 0 {
		h.setSearchAnchor(cacheKey, page, users[0].ID)
	}

	hasPrev := page > 1
	hasNext := false
	prevPage := maxInt(1, page-1)
	nextPage := page + 1
	hasJumpL100 := page > 100
	hasJumpL1k := page > 1000
	hasJumpR100 := false
	hasJumpR1k := false
	jumpL100 := maxInt(1, page-100)
	jumpL1k := maxInt(1, page-1000)
	jumpR100 := page + 100
	jumpR1k := page + 1000

	if searchMode {
		if len(users) > perPage {
			hasNext = true
			users = users[:perPage]
		}
		// Keep jump-right controls active during search mode for fast traversal.
		hasJumpR100 = true
		hasJumpR1k = true
		totalPages = page + 2
	} else {
		hasNext = page < totalPages
		hasJumpR100 = page+100 <= totalPages
		hasJumpR1k = page+1000 <= totalPages
		jumpR100 = minInt(totalPages, page+100)
		jumpR1k = minInt(totalPages, page+1000)
	}

	startPage := page - 2
	if startPage < 1 {
		startPage = 1
	}
	endPage := page + 2
	if !searchMode && endPage > totalPages {
		endPage = totalPages
	}
	pages := make([]int, 0, endPage-startPage+1)
	for i := startPage; i <= endPage; i++ {
		pages = append(pages, i)
	}

	startItem := 0
	endItem := 0
	if len(users) > 0 {
		startItem = offset + 1
		endItem = offset + len(users)
	}
	if searchMode {
		totalUsers = int64(endItem)
	}

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title":       "Dashboard",
		"name":        name,
		"role":        role,
		"isAdmin":     role == "admin",
		"users":       users,
		"error":       pickMessage(c.Query("error"), formError),
		"success":     pickMessage(formSuccess, c.Query("success")),
		"searchQ":     searchQ,
		"searchBy":    searchBy,
		"searchMode":  searchMode,
		"page":        page,
		"perPage":     perPage,
		"perPageList": allowedPerPage,
		"totalUsers":  totalUsers,
		"totalPages":  totalPages,
		"pages":       pages,
		"startItem":   startItem,
		"endItem":     endItem,
		"hasPrev":     hasPrev,
		"hasNext":     hasNext,
		"prevPage":    prevPage,
		"nextPage":    nextPage,
		"hasJumpL100": hasJumpL100,
		"hasJumpL1k":  hasJumpL1k,
		"hasJumpR100": hasJumpR100,
		"hasJumpR1k":  hasJumpR1k,
		"jumpL100":    jumpL100,
		"jumpL1k":     jumpL1k,
		"jumpR100":    jumpR100,
		"jumpR1k":     jumpR1k,
	})
}

func normalizeRole(role string) string {
	if strings.ToLower(strings.TrimSpace(role)) == "admin" {
		return "admin"
	}
	return "user"
}

func isAdmin(c *gin.Context) bool {
	role, _ := sessions.Default(c).Get("user_role").(string)
	return strings.EqualFold(strings.TrimSpace(role), "admin")
}

func sessionUserID(c *gin.Context) uint {
	raw := sessions.Default(c).Get("user_id")
	switch v := raw.(type) {
	case uint:
		return v
	case int:
		if v < 0 {
			return 0
		}
		return uint(v)
	case int64:
		if v < 0 {
			return 0
		}
		return uint(v)
	case float64:
		if v < 0 {
			return 0
		}
		return uint(v)
	default:
		return 0
	}
}

func pickMessage(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}

func getPagination(c *gin.Context) (int, int) {
	page := parsePositiveInt(firstNonEmpty(c.PostForm("page"), c.Query("page")), 1)
	perPage := normalizePerPage(parsePositiveInt(firstNonEmpty(c.PostForm("per_page"), c.Query("per_page")), defaultPerPage))
	return page, perPage
}

func getSearchQuery(c *gin.Context) string {
	return strings.TrimSpace(firstNonEmpty(c.PostForm("q"), c.Query("q")))
}

func getSearchBy(c *gin.Context) string {
	return normalizeSearchBy(firstNonEmpty(c.PostForm("search_by"), c.Query("search_by")))
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func normalizePerPage(value int) int {
	for _, allowed := range allowedPerPage {
		if value == allowed {
			return value
		}
	}
	return defaultPerPage
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func buildDashboardURL(page, perPage int, searchQ, searchBy, messageKey, messageValue string) string {
	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("per_page", strconv.Itoa(perPage))
	query.Set("search_by", normalizeSearchBy(searchBy))
	if strings.TrimSpace(searchQ) != "" {
		query.Set("q", searchQ)
	}
	if strings.TrimSpace(messageKey) != "" && strings.TrimSpace(messageValue) != "" {
		query.Set(messageKey, messageValue)
	}
	return "/dashboard?" + query.Encode()
}

func (h *DashboardHandler) countUsers(searchQ, searchBy string) (int64, error) {
	if strings.TrimSpace(searchQ) == "" {
		return h.getTotalUsers()
	}

	var total int64
	if err := applySearchFilter(h.db.Model(&models.User{}), searchQ, searchBy).
		Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func applySearchFilter(query *gorm.DB, searchQ, searchBy string) *gorm.DB {
	term := strings.TrimSpace(searchQ)
	if term == "" {
		return query
	}

	searchBy = normalizeSearchBy(searchBy)
	keyword := "%" + term + "%"

	switch searchBy {
	case "id":
		id, err := strconv.ParseUint(term, 10, 64)
		if err != nil {
			return query.Where("1 = 0")
		}
		return query.Where("id = ?", id)
	case "name":
		return query.Where("name LIKE ?", keyword)
	case "email":
		return query.Where("email LIKE ?", keyword)
	case "role":
		return query.Where("role LIKE ?", keyword)
	default:
		if id, err := strconv.ParseUint(term, 10, 64); err == nil {
			return query.Where(
				"id = ? OR name LIKE ? OR email LIKE ? OR role LIKE ?",
				id, keyword, keyword, keyword,
			)
		}
		return query.Where(
			"name LIKE ? OR email LIKE ? OR role LIKE ?",
			keyword, keyword, keyword,
		)
	}
}

func normalizeSearchBy(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if _, ok := allowedSearchBy[value]; ok {
		return value
	}
	return "all"
}

func (h *DashboardHandler) getTotalUsers() (int64, error) {
	h.mu.RLock()
	if h.totalUsersLoaded {
		value := h.cachedTotalUsers
		h.mu.RUnlock()
		return value, nil
	}
	h.mu.RUnlock()

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.totalUsersLoaded {
		return h.cachedTotalUsers, nil
	}

	// Use exact count to avoid inaccurate totals from metadata estimates.
	var exact int64
	if err := h.db.Model(&models.User{}).Count(&exact).Error; err != nil {
		return 0, err
	}

	h.cachedTotalUsers = exact
	h.totalUsersLoaded = true

	return exact, nil
}

func (h *DashboardHandler) adjustTotalUsersCache(delta int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.totalUsersLoaded {
		h.searchAnchors = make(map[string]map[int]uint)
		return
	}
	h.cachedTotalUsers += delta
	if h.cachedTotalUsers < 0 {
		h.cachedTotalUsers = 0
	}
	h.searchAnchors = make(map[string]map[int]uint)
}

func (h *DashboardHandler) invalidateTotalUsersCache() {
	h.mu.Lock()
	h.totalUsersLoaded = false
	h.searchAnchors = make(map[string]map[int]uint)
	h.mu.Unlock()
}

func (h *DashboardHandler) searchAnchorKey(searchQ, searchBy string, perPage int) string {
	return strings.ToLower(strings.TrimSpace(searchBy)) + "|" + strings.ToLower(strings.TrimSpace(searchQ)) + "|" + strconv.Itoa(perPage)
}

func (h *DashboardHandler) getNearestSearchAnchor(key string, page int) (int, uint, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	pages, ok := h.searchAnchors[key]
	if !ok || len(pages) == 0 {
		return 0, 0, false
	}

	bestPage := 0
	var bestID uint
	for p, id := range pages {
		if p <= page && p > bestPage && id > 0 {
			bestPage = p
			bestID = id
		}
	}
	if bestPage == 0 {
		return 0, 0, false
	}
	return bestPage, bestID, true
}

func (h *DashboardHandler) setSearchAnchor(key string, page int, id uint) {
	if page < 1 || id == 0 {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	pages, ok := h.searchAnchors[key]
	if !ok {
		pages = make(map[int]uint)
		h.searchAnchors[key] = pages
	}
	pages[page] = id
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
