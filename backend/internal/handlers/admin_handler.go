package handlers

import (
	"context"
	"math"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/tessera/tessera/internal/middleware"
	"github.com/tessera/tessera/internal/models"
	"github.com/tessera/tessera/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	db           *pgxpool.Pool
	userRepo     *repository.UserRepository
	fileRepo     *repository.FileRepository
	activityRepo *repository.ActivityRepository
	log          zerolog.Logger
}

func NewAdminHandler(
	db *pgxpool.Pool,
	userRepo *repository.UserRepository,
	fileRepo *repository.FileRepository,
	activityRepo *repository.ActivityRepository,
	log zerolog.Logger,
) *AdminHandler {
	return &AdminHandler{
		db:           db,
		userRepo:     userRepo,
		fileRepo:     fileRepo,
		activityRepo: activityRepo,
		log:          log,
	}
}

// SystemStats represents overall system statistics
type SystemStats struct {
	TotalUsers     int64 `json:"totalUsers"`
	ActiveUsers    int64 `json:"activeUsers"`
	TotalStorage   int64 `json:"totalStorage"`
	UsedStorage    int64 `json:"usedStorage"`
	TotalFiles     int64 `json:"totalFiles"`
	TotalShares    int64 `json:"totalShares"`
	UploadsToday   int64 `json:"uploadsToday"`
	DownloadsToday int64 `json:"downloadsToday"`
}

// SystemSettings represents configurable system settings
type SystemSettings struct {
	SiteName                 string   `json:"siteName"`
	SiteURL                  string   `json:"siteUrl"`
	DefaultQuota             int64    `json:"defaultQuota"`
	AllowRegistration        bool     `json:"allowRegistration"`
	RequireEmailVerification bool     `json:"requireEmailVerification"`
	MaxUploadSize            int64    `json:"maxUploadSize"`
	AllowedFileTypes         []string `json:"allowedFileTypes"`
	MaintenanceMode          bool     `json:"maintenanceMode"`
	SMTPHost                 string   `json:"smtpHost"`
	SMTPPort                 int      `json:"smtpPort"`
	SMTPUser                 string   `json:"smtpUser"`
	SMTPFrom                 string   `json:"smtpFrom"`
}

// AdminUserResponse represents a user for admin display
type AdminUserResponse struct {
	ID            uuid.UUID  `json:"id"`
	Email         string     `json:"email"`
	Name          string     `json:"name"`
	Role          string     `json:"role"`
	StorageUsed   int64      `json:"storageUsed"`
	StorageQuota  int64      `json:"storageQuota"`
	CreatedAt     time.Time  `json:"createdAt"`
	LastLoginAt   *time.Time `json:"lastLoginAt"`
	IsActive      bool       `json:"isActive"`
	EmailVerified bool       `json:"emailVerified"`
}

// GetStats returns system statistics
func (h *AdminHandler) GetStats(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	var totalUsers, activeUsers int64
	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&totalUsers)
	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE is_active = true").Scan(&activeUsers)

	var totalFiles int64
	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM files WHERE is_folder = false").Scan(&totalFiles)

	var totalShares int64
	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM shares").Scan(&totalShares)

	var usedStorage int64
	h.db.QueryRow(ctx, "SELECT COALESCE(SUM(size), 0) FROM files").Scan(&usedStorage)

	// Get today's activity
	today := time.Now().Truncate(24 * time.Hour)
	var uploadsToday, downloadsToday int64
	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM activities WHERE action = 'upload' AND created_at >= $1", today).Scan(&uploadsToday)
	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM activities WHERE action = 'download' AND created_at >= $1", today).Scan(&downloadsToday)

	// Assume 1TB total storage for now (could be configurable)
	totalStorage := int64(1024 * 1024 * 1024 * 1024) // 1TB

	stats := SystemStats{
		TotalUsers:     totalUsers,
		ActiveUsers:    activeUsers,
		TotalStorage:   totalStorage,
		UsedStorage:    usedStorage,
		TotalFiles:     totalFiles,
		TotalShares:    totalShares,
		UploadsToday:   uploadsToday,
		DownloadsToday: downloadsToday,
	}

	return c.JSON(stats)
}

// GetSettings returns system settings
func (h *AdminHandler) GetSettings(c *fiber.Ctx) error {
	settings := SystemSettings{
		SiteName:                 "Tessera",
		SiteURL:                  "http://localhost:3000",
		DefaultQuota:             10 * 1024 * 1024 * 1024,
		AllowRegistration:        true,
		RequireEmailVerification: true,
		MaxUploadSize:            100 * 1024 * 1024,
		AllowedFileTypes:         []string{"*"},
		MaintenanceMode:          false,
		SMTPHost:                 "",
		SMTPPort:                 587,
		SMTPUser:                 "",
		SMTPFrom:                 "",
	}
	return c.JSON(settings)
}

// UpdateSettings updates system settings
func (h *AdminHandler) UpdateSettings(c *fiber.Ctx) error {
	var input SystemSettings
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	return c.JSON(input)
}

// ListUsers returns paginated list of users
func (h *AdminHandler) ListUsers(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	search := c.Query("search", "")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int64
	var query string
	var args []interface{}

	if search != "" {
		searchPattern := "%" + search + "%"
		h.db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE name ILIKE $1 OR email ILIKE $1", searchPattern).Scan(&total)
		query = `
			SELECT id, email, name, role, storage_limit, is_active, created_at, last_login_at
			FROM users
			WHERE name ILIKE $1 OR email ILIKE $1
			ORDER BY created_at DESC
			OFFSET $2 LIMIT $3
		`
		args = []interface{}{searchPattern, offset, limit}
	} else {
		h.db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&total)
		query = `
			SELECT id, email, name, role, storage_limit, is_active, created_at, last_login_at
			FROM users
			ORDER BY created_at DESC
			OFFSET $1 LIMIT $2
		`
		args = []interface{}{offset, limit}
	}

	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to fetch users")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch users",
		})
	}
	defer rows.Close()

	var users []AdminUserResponse
	for rows.Next() {
		var user AdminUserResponse
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Name,
			&user.Role,
			&user.StorageQuota,
			&user.IsActive,
			&user.CreatedAt,
			&user.LastLoginAt,
		); err != nil {
			h.log.Error().Err(err).Msg("Failed to scan user")
			continue
		}
		user.EmailVerified = true // Assume verified

		// Get storage used
		var storageUsed int64
		h.db.QueryRow(ctx, "SELECT COALESCE(SUM(size), 0) FROM files WHERE owner_id = $1", user.ID).Scan(&storageUsed)
		user.StorageUsed = storageUsed

		users = append(users, user)
	}

	return c.JSON(fiber.Map{
		"users":      users,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"totalPages": int(math.Ceil(float64(total) / float64(limit))),
	})
}

// GetUser returns a single user by ID
func (h *AdminHandler) GetUser(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	var storageUsed int64
	h.db.QueryRow(ctx, "SELECT COALESCE(SUM(size), 0) FROM files WHERE owner_id = $1", user.ID).Scan(&storageUsed)

	response := AdminUserResponse{
		ID:            user.ID,
		Email:         user.Email,
		Name:          user.Name,
		Role:          user.Role,
		StorageUsed:   storageUsed,
		StorageQuota:  user.StorageLimit,
		CreatedAt:     user.CreatedAt,
		LastLoginAt:   user.LastLoginAt,
		IsActive:      user.IsActive,
		EmailVerified: true,
	}

	return c.JSON(response)
}

// CreateUser creates a new user (admin only)
func (h *AdminHandler) CreateUser(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	var input struct {
		Email        string `json:"email"`
		Password     string `json:"password"`
		Name         string `json:"name"`
		Role         string `json:"role"`
		StorageQuota int64  `json:"storageQuota"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if input.Email == "" || input.Password == "" || input.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email, password, and name are required",
		})
	}

	// Validate role
	if input.Role == "" {
		input.Role = "user"
	}
	if input.Role != "admin" && input.Role != "user" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid role. Must be 'admin' or 'user'",
		})
	}

	// Check if email already exists
	exists, err := h.userRepo.EmailExists(ctx, input.Email)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to check email existence")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}
	if exists {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Email already registered",
		})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to hash password")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}

	// Default storage quota: 10GB
	storageQuota := input.StorageQuota
	if storageQuota <= 0 {
		storageQuota = 10 * 1024 * 1024 * 1024
	}

	user := &models.User{
		Email:        input.Email,
		PasswordHash: string(hashedPassword),
		Name:         input.Name,
		Role:         input.Role,
		StorageLimit: storageQuota,
		IsActive:     true,
	}

	if err := h.userRepo.Create(ctx, user); err != nil {
		h.log.Error().Err(err).Msg("Failed to create user")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}

	response := AdminUserResponse{
		ID:            user.ID,
		Email:         user.Email,
		Name:          user.Name,
		Role:          user.Role,
		StorageUsed:   0,
		StorageQuota:  user.StorageLimit,
		CreatedAt:     user.CreatedAt,
		LastLoginAt:   nil,
		IsActive:      user.IsActive,
		EmailVerified: true,
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

// UpdateUser updates a user
func (h *AdminHandler) UpdateUser(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	currentUserID := middleware.GetUserID(c)
	targetUserID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var input struct {
		Name         *string `json:"name"`
		Role         *string `json:"role"`
		StorageQuota *int64  `json:"storageQuota"`
		IsActive     *bool   `json:"isActive"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Prevent admins from demoting or disabling themselves
	if currentUserID == targetUserID {
		if input.Role != nil && *input.Role != "admin" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Cannot demote yourself. Ask another admin to change your role.",
			})
		}
		if input.IsActive != nil && !*input.IsActive {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Cannot disable your own account.",
			})
		}
	}

	user, err := h.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	if input.Name != nil {
		user.Name = *input.Name
	}
	if input.Role != nil {
		if *input.Role != "admin" && *input.Role != "user" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid role",
			})
		}
		user.Role = *input.Role
	}
	if input.StorageQuota != nil {
		user.StorageLimit = *input.StorageQuota
	}
	if input.IsActive != nil {
		user.IsActive = *input.IsActive
	}

	if err := h.userRepo.Update(ctx, user); err != nil {
		h.log.Error().Err(err).Msg("Failed to update user")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user",
		})
	}

	var storageUsed int64
	h.db.QueryRow(ctx, "SELECT COALESCE(SUM(size), 0) FROM files WHERE owner_id = $1", user.ID).Scan(&storageUsed)

	response := AdminUserResponse{
		ID:            user.ID,
		Email:         user.Email,
		Name:          user.Name,
		Role:          user.Role,
		StorageUsed:   storageUsed,
		StorageQuota:  user.StorageLimit,
		CreatedAt:     user.CreatedAt,
		LastLoginAt:   user.LastLoginAt,
		IsActive:      user.IsActive,
		EmailVerified: true,
	}

	return c.JSON(response)
}

// DeleteUser deletes a user and their files
func (h *AdminHandler) DeleteUser(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	currentUserID := middleware.GetUserID(c)
	targetUserID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Prevent admin from deleting themselves
	if currentUserID == targetUserID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot delete your own account. Ask another admin to delete it.",
		})
	}

	_, err = h.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Use transaction for atomic deletion
	tx, err := h.db.Begin(ctx)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to begin transaction")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user",
		})
	}
	defer tx.Rollback(ctx)

	// Delete user's related data in order
	if _, err := tx.Exec(ctx, "DELETE FROM file_versions WHERE file_id IN (SELECT id FROM files WHERE owner_id = $1)", targetUserID); err != nil {
		h.log.Error().Err(err).Msg("Failed to delete user file versions")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user data",
		})
	}

	if _, err := tx.Exec(ctx, "DELETE FROM shares WHERE owner_id = $1", targetUserID); err != nil {
		h.log.Error().Err(err).Msg("Failed to delete user shares")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user data",
		})
	}

	if _, err := tx.Exec(ctx, "DELETE FROM files WHERE owner_id = $1", targetUserID); err != nil {
		h.log.Error().Err(err).Msg("Failed to delete user files")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user data",
		})
	}

	if _, err := tx.Exec(ctx, "DELETE FROM activities WHERE user_id = $1", targetUserID); err != nil {
		h.log.Error().Err(err).Msg("Failed to delete user activities")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user data",
		})
	}

	if _, err := tx.Exec(ctx, "DELETE FROM sessions WHERE user_id = $1", targetUserID); err != nil {
		h.log.Error().Err(err).Msg("Failed to delete user sessions")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user data",
		})
	}

	if _, err := tx.Exec(ctx, "DELETE FROM users WHERE id = $1", targetUserID); err != nil {
		h.log.Error().Err(err).Msg("Failed to delete user")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user",
		})
	}

	if err := tx.Commit(ctx); err != nil {
		h.log.Error().Err(err).Msg("Failed to commit transaction")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ActivityLogResponse represents an activity log entry
type ActivityLogResponse struct {
	ID           uuid.UUID              `json:"id"`
	UserID       uuid.UUID              `json:"userId"`
	UserEmail    string                 `json:"userEmail"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resourceType"`
	ResourceID   string                 `json:"resourceId"`
	IPAddress    string                 `json:"ipAddress"`
	UserAgent    string                 `json:"userAgent"`
	CreatedAt    time.Time              `json:"createdAt"`
	Details      map[string]interface{} `json:"details"`
}

// GetActivityLogs returns paginated activity logs
func (h *AdminHandler) GetActivityLogs(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	action := c.Query("action", "")
	userIDStr := c.Query("userId", "")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	var total int64

	baseQuery := "FROM activities a LEFT JOIN users u ON a.user_id = u.id WHERE 1=1"
	args := []interface{}{}
	argNum := 1

	if action != "" {
		baseQuery += " AND a.action = $" + strconv.Itoa(argNum)
		args = append(args, action)
		argNum++
	}
	if userIDStr != "" {
		uid, err := uuid.Parse(userIDStr)
		if err == nil {
			baseQuery += " AND a.user_id = $" + strconv.Itoa(argNum)
			args = append(args, uid)
			argNum++
		}
	}

	countQuery := "SELECT COUNT(*) " + baseQuery
	h.db.QueryRow(ctx, countQuery, args...).Scan(&total)

	selectQuery := `
		SELECT a.id, a.user_id, COALESCE(u.email, '') as user_email, a.action, a.resource_type, a.resource_id, 
		       COALESCE(a.ip_address, '') as ip_address, COALESCE(a.user_agent, '') as user_agent, a.created_at
		` + baseQuery + ` ORDER BY a.created_at DESC OFFSET $` + strconv.Itoa(argNum) + ` LIMIT $` + strconv.Itoa(argNum+1)

	args = append(args, offset, limit)

	rows, err := h.db.Query(ctx, selectQuery, args...)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to fetch activity logs")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch activity logs",
		})
	}
	defer rows.Close()

	var logs []ActivityLogResponse
	for rows.Next() {
		var log ActivityLogResponse
		if err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.UserEmail,
			&log.Action,
			&log.ResourceType,
			&log.ResourceID,
			&log.IPAddress,
			&log.UserAgent,
			&log.CreatedAt,
		); err != nil {
			h.log.Error().Err(err).Msg("Failed to scan activity log")
			continue
		}
		logs = append(logs, log)
	}

	return c.JSON(fiber.Map{
		"logs":       logs,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"totalPages": int(math.Ceil(float64(total) / float64(limit))),
	})
}

// ClearCache clears application cache
func (h *AdminHandler) ClearCache(c *fiber.Ctx) error {
	h.log.Info().Msg("Cache cleared by admin")
	return c.JSON(fiber.Map{
		"message": "Cache cleared successfully",
	})
}

// RunCleanup runs cleanup tasks
func (h *AdminHandler) RunCleanup(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 60*time.Second)
	defer cancel()

	result, err := h.db.Exec(ctx, "DELETE FROM shares WHERE expires_at IS NOT NULL AND expires_at < $1", time.Now())
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to cleanup expired shares")
	}

	sharesExpired := result.RowsAffected()

	h.log.Info().Int64("sharesExpired", sharesExpired).Msg("Cleanup completed")

	return c.JSON(fiber.Map{
		"message":       "Cleanup completed",
		"filesRemoved":  0,
		"sharesExpired": sharesExpired,
	})
}
