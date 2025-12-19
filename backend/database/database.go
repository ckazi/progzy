package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	pq "github.com/lib/pq"
	"proxy-server/models"

	"golang.org/x/crypto/bcrypt"
)

type Database struct {
	DB *sql.DB
}

const defaultLogRetentionDays = 30

func NewDatabase() (*Database, error) {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var db *sql.DB
	var err error

	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Printf("Failed to connect to database, retrying in 2 seconds... (%d/10)", i+1)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	database := &Database{DB: db}

	if err := database.ensureProxySchema(); err != nil {
		return nil, fmt.Errorf("failed to ensure proxy schema: %v", err)
	}

	log.Println("Successfully connected to database")

	return database, nil
}

func (d *Database) Close() error {
	return d.DB.Close()
}

func (d *Database) IsInitialized() (bool, error) {
	var count int
	err := d.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (d *Database) CreateUser(user *models.UserCreate, passwordHash string) (*models.User, error) {
	proxyType := user.ProxyType
	if proxyType == "" {
		proxyType = "default"
	}
	var newUser models.User
	err := d.DB.QueryRow(`
		INSERT INTO users (username, password_hash, email, comment, is_admin, proxy_type)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, username, email, comment, is_admin, is_active, proxy_type, twofa_enabled, created_at, updated_at
	`, user.Username, passwordHash, user.Email, user.Comment, user.IsAdmin, proxyType).
		Scan(&newUser.ID, &newUser.Username, &newUser.Email, &newUser.Comment,
			&newUser.IsAdmin, &newUser.IsActive, &newUser.ProxyType, &newUser.TwoFAEnabled, &newUser.CreatedAt, &newUser.UpdatedAt)

	if err != nil {
		return nil, err
	}
	return &newUser, nil
}

func (d *Database) GetUserByUsername(username string) (*models.User, error) {
	return d.GetUserByUsernameCtx(context.Background(), username)
}

func (d *Database) GetUserByUsernameCtx(ctx context.Context, username string) (*models.User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}
	var user models.User
	err := d.DB.QueryRowContext(ctx, `
		SELECT id, username, password_hash, email, comment, is_admin, is_active, proxy_type, twofa_enabled, created_at, updated_at
		FROM users
		WHERE LOWER(username) = LOWER($1)
		ORDER BY id
		LIMIT 1
	`, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Email,
		&user.Comment, &user.IsAdmin, &user.IsActive, &user.ProxyType, &user.TwoFAEnabled, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *Database) GetUserByID(id int) (*models.User, error) {
	var user models.User
	err := d.DB.QueryRow(`
		SELECT id, username, password_hash, email, comment, is_admin, is_active, proxy_type, twofa_enabled, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Email,
		&user.Comment, &user.IsAdmin, &user.IsActive, &user.ProxyType, &user.TwoFAEnabled, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	user.Whitelist, _ = d.getProxyList("user_proxy_whitelist", user.ID)
	user.Blacklist, _ = d.getProxyList("user_proxy_blacklist", user.ID)
	return &user, nil
}

func (d *Database) GetAllUsers() ([]models.User, error) {
	rows, err := d.DB.Query(`
		SELECT u.id, u.username, u.email, u.comment, u.is_admin, u.is_active,
		       u.proxy_type, u.twofa_enabled, u.created_at, u.updated_at,
		       COALESCE((
		       	SELECT ARRAY_AGG(value ORDER BY id)
		       	FROM user_proxy_whitelist
		       	WHERE user_id = u.id
		       ), ARRAY[]::text[]) AS whitelist,
		       COALESCE((
		       	SELECT ARRAY_AGG(value ORDER BY id)
		       	FROM user_proxy_blacklist
		       	WHERE user_id = u.id
		       ), ARRAY[]::text[]) AS blacklist
		FROM users u
		ORDER BY u.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		var whitelist []string
		var blacklist []string
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.Comment,
			&user.IsAdmin, &user.IsActive, &user.ProxyType, &user.TwoFAEnabled,
			&user.CreatedAt, &user.UpdatedAt, pq.Array(&whitelist), pq.Array(&blacklist))
		if err != nil {
			return nil, err
		}
		user.Whitelist = append([]string(nil), whitelist...)
		user.Blacklist = append([]string(nil), blacklist...)
		users = append(users, user)
	}
	return users, nil
}

func (d *Database) UpdateUser(id int, update *models.UserUpdate) error {
	query := "UPDATE users SET "
	args := []interface{}{}
	argCount := 1

	if update.Email != nil {
		query += fmt.Sprintf("email = $%d, ", argCount)
		args = append(args, *update.Email)
		argCount++
	}
	if update.Comment != nil {
		query += fmt.Sprintf("comment = $%d, ", argCount)
		args = append(args, *update.Comment)
		argCount++
	}
	if update.IsAdmin != nil {
		query += fmt.Sprintf("is_admin = $%d, ", argCount)
		args = append(args, *update.IsAdmin)
		argCount++
	}
	if update.IsActive != nil {
		query += fmt.Sprintf("is_active = $%d, ", argCount)
		args = append(args, *update.IsActive)
		argCount++
	}
	if update.ProxyType != nil {
		query += fmt.Sprintf("proxy_type = $%d, ", argCount)
		args = append(args, *update.ProxyType)
		argCount++
	}
	if update.Password != nil {
		query += fmt.Sprintf("password_hash = $%d, ", argCount)
		args = append(args, *update.Password)
		argCount++
	}

	if len(args) == 0 {
		return fmt.Errorf("no fields to update")
	}

	query = query[:len(query)-2]
	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, id)

	_, err := d.DB.Exec(query, args...)
	if err != nil {
		return err
	}

	if update.Whitelist != nil {
		if _, err := d.replaceProxyList("user_proxy_whitelist", id, *update.Whitelist); err != nil {
			return err
		}
	}
	if update.Blacklist != nil {
		if _, err := d.replaceProxyList("user_proxy_blacklist", id, *update.Blacklist); err != nil {
			return err
		}
	}
	return nil
}

func (d *Database) DeleteUser(id int) error {
	_, err := d.DB.Exec("DELETE FROM users WHERE id = $1", id)
	return err
}

func (d *Database) GetProxySettings() ([]models.ProxySetting, error) {
	rows, err := d.DB.Query(`
		SELECT id, key, value, description, updated_at
		FROM proxy_settings ORDER BY key
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []models.ProxySetting
	for rows.Next() {
		var setting models.ProxySetting
		err := rows.Scan(&setting.ID, &setting.Key, &setting.Value, &setting.Description, &setting.UpdatedAt)
		if err != nil {
			return nil, err
		}
		settings = append(settings, setting)
	}
	return settings, nil
}

func (d *Database) GetProxySetting(key string) (*models.ProxySetting, error) {
	var setting models.ProxySetting
	err := d.DB.QueryRow(`
		SELECT id, key, value, description, updated_at
		FROM proxy_settings
		WHERE key = $1
	`, key).Scan(&setting.ID, &setting.Key, &setting.Value, &setting.Description, &setting.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func (d *Database) UpdateProxySetting(key, value string) error {
	_, err := d.DB.Exec(`
		UPDATE proxy_settings SET value = $1, updated_at = CURRENT_TIMESTAMP
		WHERE key = $2
	`, value, key)
	return err
}

func (d *Database) LogRequest(log *models.RequestLog) error {
	_, err := d.DB.Exec(`
		INSERT INTO request_logs (user_id, method, url, status_code, bytes_sent, bytes_received, duration_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, log.UserID, log.Method, log.URL, log.StatusCode, log.BytesSent, log.BytesReceived, log.DurationMs)
	return err
}

func (d *Database) UpdateTrafficStats(userID int, bytesSent, bytesReceived int64) error {
	_, err := d.DB.Exec(`
		INSERT INTO traffic_stats (user_id, bytes_sent, bytes_received, request_count, date)
		VALUES ($1, $2, $3, 1, CURRENT_DATE)
		ON CONFLICT (user_id, date)
		DO UPDATE SET
			bytes_sent = traffic_stats.bytes_sent + $2,
			bytes_received = traffic_stats.bytes_received + $3,
			request_count = traffic_stats.request_count + 1
	`, userID, bytesSent, bytesReceived)
	return err
}

func (d *Database) GetTrafficStats(limit int, startDate, endDate *time.Time) ([]models.TrafficStats, error) {
	query := `
		SELECT ts.id, ts.user_id, u.username, ts.bytes_sent, ts.bytes_received,
		       ts.request_count, ts.date
		FROM traffic_stats ts
		JOIN users u ON ts.user_id = u.id
	`

	where := []string{}
	args := []interface{}{}
	argPos := 1

	if startDate != nil {
		where = append(where, fmt.Sprintf("ts.date >= $%d", argPos))
		args = append(args, *startDate)
		argPos++
	}
	if endDate != nil {
		where = append(where, fmt.Sprintf("ts.date <= $%d", argPos))
		args = append(args, *endDate)
		argPos++
	}

	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	query += " ORDER BY ts.date DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := d.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.TrafficStats
	for rows.Next() {
		var stat models.TrafficStats
		err := rows.Scan(&stat.ID, &stat.UserID, &stat.Username, &stat.BytesSent,
			&stat.BytesReceived, &stat.RequestCount, &stat.Date)
		if err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}
	return stats, nil
}

func (d *Database) GetRequestLogs(filters *models.LogFilterOptions) ([]models.RequestLog, error) {
	baseQuery := `
		SELECT rl.id, rl.user_id, COALESCE(u.username, 'unknown'), rl.method, rl.url,
		       rl.status_code, rl.bytes_sent, rl.bytes_received, rl.duration_ms, rl.created_at
		FROM request_logs rl
		LEFT JOIN users u ON rl.user_id = u.id
	`

	where := []string{}
	args := []interface{}{}
	argPos := 1

	if filters != nil {
		if filters.StartDate != nil {
			where = append(where, fmt.Sprintf("rl.created_at >= $%d", argPos))
			args = append(args, *filters.StartDate)
			argPos++
		}
		if filters.EndDate != nil {
			where = append(where, fmt.Sprintf("rl.created_at <= $%d", argPos))
			args = append(args, *filters.EndDate)
			argPos++
		}
		if filters.UserID != nil {
			where = append(where, fmt.Sprintf("rl.user_id = $%d", argPos))
			args = append(args, *filters.UserID)
			argPos++
		}
		if filters.Username != "" {
			where = append(where, fmt.Sprintf("LOWER(u.username) LIKE LOWER($%d)", argPos))
			args = append(args, "%"+filters.Username+"%")
			argPos++
		}
		if filters.Method != "" {
			where = append(where, fmt.Sprintf("LOWER(rl.method) = LOWER($%d)", argPos))
			args = append(args, filters.Method)
			argPos++
		}
		if filters.URLContains != "" {
			where = append(where, fmt.Sprintf("rl.url ILIKE $%d", argPos))
			args = append(args, "%"+filters.URLContains+"%")
			argPos++
		}
		if filters.StatusCode != nil {
			where = append(where, fmt.Sprintf("rl.status_code = $%d", argPos))
			args = append(args, *filters.StatusCode)
			argPos++
		}
		if filters.MinBytesSent != nil {
			where = append(where, fmt.Sprintf("rl.bytes_sent >= $%d", argPos))
			args = append(args, *filters.MinBytesSent)
			argPos++
		}
		if filters.MaxBytesSent != nil {
			where = append(where, fmt.Sprintf("rl.bytes_sent <= $%d", argPos))
			args = append(args, *filters.MaxBytesSent)
			argPos++
		}
		if filters.MinBytesRecv != nil {
			where = append(where, fmt.Sprintf("rl.bytes_received >= $%d", argPos))
			args = append(args, *filters.MinBytesRecv)
			argPos++
		}
		if filters.MaxBytesRecv != nil {
			where = append(where, fmt.Sprintf("rl.bytes_received <= $%d", argPos))
			args = append(args, *filters.MaxBytesRecv)
			argPos++
		}
		if filters.MinDurationMs != nil {
			where = append(where, fmt.Sprintf("rl.duration_ms >= $%d", argPos))
			args = append(args, *filters.MinDurationMs)
			argPos++
		}
		if filters.MaxDurationMs != nil {
			where = append(where, fmt.Sprintf("rl.duration_ms <= $%d", argPos))
			args = append(args, *filters.MaxDurationMs)
			argPos++
		}
	}

	if len(where) > 0 {
		baseQuery += " WHERE " + strings.Join(where, " AND ")
	}

	sortColumns := map[string]string{
		"created_at":     "rl.created_at",
		"username":       "u.username",
		"method":         "rl.method",
		"url":            "rl.url",
		"status_code":    "rl.status_code",
		"bytes_sent":     "rl.bytes_sent",
		"bytes_received": "rl.bytes_received",
		"duration_ms":    "rl.duration_ms",
	}

	orderBy := "rl.created_at"
	if filters != nil {
		if col, ok := sortColumns[filters.SortBy]; ok {
			orderBy = col
		}
	}

	orderDirection := "DESC"
	if filters != nil && strings.EqualFold(filters.SortOrder, "asc") {
		orderDirection = "ASC"
	}

	baseQuery += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDirection)

	if filters != nil && filters.Limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT %d", filters.Limit)
		if filters.Offset > 0 {
			baseQuery += fmt.Sprintf(" OFFSET %d", filters.Offset)
		}
	}

	rows, err := d.DB.Query(baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.RequestLog
	for rows.Next() {
		var log models.RequestLog
		err := rows.Scan(&log.ID, &log.UserID, &log.Username, &log.Method, &log.URL,
			&log.StatusCode, &log.BytesSent, &log.BytesReceived, &log.DurationMs, &log.CreatedAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, nil
}

func (d *Database) GetDashboardStats() (*models.StatsResponse, error) {
	stats := &models.StatsResponse{}

	err := d.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	if err != nil {
		return nil, err
	}

	err = d.DB.QueryRow("SELECT COUNT(*) FROM users WHERE is_active = true").Scan(&stats.ActiveUsers)
	if err != nil {
		return nil, err
	}

	err = d.DB.QueryRow("SELECT COALESCE(SUM(request_count), 0) FROM traffic_stats").Scan(&stats.TotalRequests)
	if err != nil {
		return nil, err
	}

	err = d.DB.QueryRow("SELECT COALESCE(SUM(bytes_sent), 0) FROM traffic_stats").Scan(&stats.TotalBytesSent)
	if err != nil {
		return nil, err
	}

	err = d.DB.QueryRow("SELECT COALESCE(SUM(bytes_received), 0) FROM traffic_stats").Scan(&stats.TotalBytesRecv)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func (d *Database) ensureProxySchema() error {
	statements := []string{
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS proxy_type VARCHAR(20) NOT NULL DEFAULT 'default'`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS twofa_secret TEXT`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS twofa_enabled BOOLEAN NOT NULL DEFAULT false`,
		`CREATE TABLE IF NOT EXISTS user_proxy_whitelist (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			value TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, value)
		)`,
		`CREATE TABLE IF NOT EXISTS user_proxy_blacklist (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			value TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, value)
		)`,
		`CREATE TABLE IF NOT EXISTS admin_audit_logs (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
			action TEXT NOT NULL,
			details TEXT,
			ip_address TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS user_twofa_backup_codes (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			code_hash TEXT NOT NULL,
			used BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			used_at TIMESTAMP NULL
		)`,
		`CREATE TABLE IF NOT EXISTS twofa_logs (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			ip_address TEXT,
			event TEXT,
			method TEXT,
			success BOOLEAN DEFAULT FALSE,
			message TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, stmt := range statements {
		if _, err := d.DB.Exec(stmt); err != nil {
			return err
		}
	}
	if _, err := d.DB.Exec(`CREATE INDEX IF NOT EXISTS idx_twofa_logs_user ON twofa_logs(user_id, created_at)`); err != nil {
		return err
	}
	if _, err := d.DB.Exec(`CREATE INDEX IF NOT EXISTS idx_twofa_logs_ip ON twofa_logs(ip_address, created_at)`); err != nil {
		return err
	}
	return nil
}

func (d *Database) GetLogRetentionDays() (int, error) {
	var value string
	err := d.DB.QueryRow("SELECT value FROM proxy_settings WHERE key = $1", "log_retention_days").Scan(&value)
	if err == sql.ErrNoRows {
		if err := d.SetLogRetentionDays(defaultLogRetentionDays); err != nil {
			return defaultLogRetentionDays, err
		}
		return defaultLogRetentionDays, nil
	}
	if err != nil {
		return defaultLogRetentionDays, err
	}

	days, err := strconv.Atoi(value)
	if err != nil || days <= 0 {
		return defaultLogRetentionDays, nil
	}
	return days, nil
}

func (d *Database) SetLogRetentionDays(days int) error {
	value := strconv.Itoa(days)
	_, err := d.DB.Exec(`
		INSERT INTO proxy_settings (key, value, description)
		VALUES ($1, $2, 'Days to retain request logs')
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = CURRENT_TIMESTAMP
	`, "log_retention_days", value)
	return err
}

func (d *Database) DeleteRequestLogsOlderThan(days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days)
	result, err := d.DB.Exec("DELETE FROM request_logs WHERE created_at < $1", cutoff)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return rows, nil
}

func sanitizeEntries(entries []string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, entry := range entries {
		normalized := strings.ToLower(strings.TrimSpace(entry))
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func (d *Database) replaceProxyList(table string, userID int, entries []string) ([]string, error) {
	sanitized := sanitizeEntries(entries)

	tx, err := d.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE user_id = $1", table), userID); err != nil {
		return nil, err
	}

	if len(sanitized) > 0 {
		insertStmt := fmt.Sprintf("INSERT INTO %s (user_id, value) VALUES ($1, $2)", table)
		for _, value := range sanitized {
			if _, err := tx.Exec(insertStmt, userID, value); err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return sanitized, nil
}

func (d *Database) getProxyList(table string, userID int) ([]string, error) {
	rows, err := d.DB.Query(fmt.Sprintf("SELECT value FROM %s WHERE user_id = $1 ORDER BY id", table), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var values []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (d *Database) SetUserProxyLists(userID int, whitelist, blacklist []string) (wl []string, bl []string, err error) {
	if wl, err = d.replaceProxyList("user_proxy_whitelist", userID, whitelist); err != nil {
		return nil, nil, err
	}
	if bl, err = d.replaceProxyList("user_proxy_blacklist", userID, blacklist); err != nil {
		return nil, nil, err
	}
	return wl, bl, nil
}

func (d *Database) GetUserProxySettings(userID int) (*models.UserProxySettings, error) {
	settings := &models.UserProxySettings{}
	err := d.DB.QueryRow("SELECT proxy_type FROM users WHERE id = $1", userID).Scan(&settings.ProxyType)
	if err != nil {
		return nil, err
	}

	var errWL, errBL error
	settings.Whitelist, errWL = d.getProxyList("user_proxy_whitelist", userID)
	settings.Blacklist, errBL = d.getProxyList("user_proxy_blacklist", userID)
	if errWL != nil {
		return nil, errWL
	}
	if errBL != nil {
		return nil, errBL
	}
	return settings, nil
}

func (d *Database) LogAdminAction(userID *int, action, details, ip string) {
	var dbUserID sql.NullInt64
	if userID != nil {
		dbUserID = sql.NullInt64{
			Int64: int64(*userID),
			Valid: true,
		}
	}

	_, err := d.DB.Exec(`
		INSERT INTO admin_audit_logs (user_id, action, details, ip_address)
		VALUES ($1, $2, $3, $4)
	`, dbUserID, action, details, ip)
	if err != nil {
		log.Printf("Failed to log admin action: %v", err)
	}
}

func (d *Database) GetAuditLogs(filters *models.AuditLogFilterOptions) ([]models.AdminAuditLog, error) {
	if filters == nil {
		filters = &models.AuditLogFilterOptions{}
	}

	query := `
		SELECT l.id, l.user_id, COALESCE(u.username, ''), l.action, l.details, l.ip_address, l.created_at
		FROM admin_audit_logs l
		LEFT JOIN users u ON l.user_id = u.id
	`

	where := []string{}
	args := []interface{}{}
	argPos := 1

	if filters.StartDate != nil {
		where = append(where, fmt.Sprintf("l.created_at >= $%d", argPos))
		args = append(args, *filters.StartDate)
		argPos++
	}
	if filters.EndDate != nil {
		where = append(where, fmt.Sprintf("l.created_at <= $%d", argPos))
		args = append(args, *filters.EndDate)
		argPos++
	}
	if filters.Username != "" {
		where = append(where, fmt.Sprintf("LOWER(u.username) LIKE LOWER($%d)", argPos))
		args = append(args, "%"+filters.Username+"%")
		argPos++
	}
	if filters.Action != "" {
		where = append(where, fmt.Sprintf("LOWER(l.action) LIKE LOWER($%d)", argPos))
		args = append(args, "%"+filters.Action+"%")
		argPos++
	}
	if filters.Details != "" {
		where = append(where, fmt.Sprintf("LOWER(l.details) LIKE LOWER($%d)", argPos))
		args = append(args, "%"+filters.Details+"%")
		argPos++
	}
	if filters.IPAddress != "" {
		where = append(where, fmt.Sprintf("LOWER(l.ip_address) LIKE LOWER($%d)", argPos))
		args = append(args, "%"+filters.IPAddress+"%")
		argPos++
	}

	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	sortColumns := map[string]string{
		"created_at": "l.created_at",
		"username":   "u.username",
		"action":     "l.action",
		"details":    "l.details",
		"ip_address": "l.ip_address",
	}

	orderBy := "l.created_at"
	if col, ok := sortColumns[strings.ToLower(filters.SortBy)]; ok {
		orderBy = col
	}

	orderDirection := "DESC"
	if strings.EqualFold(filters.SortOrder, "asc") {
		orderDirection = "ASC"
	}

	query += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDirection)

	limit := filters.Limit
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filters.Offset)
	}

	rows, err := d.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.AdminAuditLog
	for rows.Next() {
		var logEntry models.AdminAuditLog
		if err := rows.Scan(&logEntry.ID, &logEntry.UserID, &logEntry.Username, &logEntry.Action, &logEntry.Details, &logEntry.IPAddress, &logEntry.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, logEntry)
	}
	return logs, nil
}

func (d *Database) StoreTwoFASecret(ctx context.Context, userID int, encryptedSecret string) error {
	_, err := d.DB.ExecContext(ctx, `
		UPDATE users
		SET twofa_secret = $1, twofa_enabled = false
		WHERE id = $2
	`, encryptedSecret, userID)
	return err
}

func (d *Database) SetTwoFAEnabled(ctx context.Context, userID int, enabled bool) error {
	_, err := d.DB.ExecContext(ctx, `
		UPDATE users
		SET twofa_enabled = $1
		WHERE id = $2
	`, enabled, userID)
	return err
}

func (d *Database) ClearTwoFAData(ctx context.Context, userID int) error {
	tx, err := d.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE users
		SET twofa_secret = NULL, twofa_enabled = false
		WHERE id = $1
	`, userID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM user_twofa_backup_codes WHERE user_id = $1`, userID); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *Database) ReplaceBackupCodes(ctx context.Context, userID int, hashes []string) error {
	tx, err := d.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM user_twofa_backup_codes WHERE user_id = $1`, userID); err != nil {
		return err
	}

	if len(hashes) > 0 {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO user_twofa_backup_codes (user_id, code_hash)
			VALUES ($1, $2)
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, hash := range hashes {
			if _, err := stmt.ExecContext(ctx, userID, hash); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (d *Database) GetTwoFASecret(ctx context.Context, userID int) (string, bool, error) {
	var secret sql.NullString
	var enabled bool
	err := d.DB.QueryRowContext(ctx, `
		SELECT twofa_secret, twofa_enabled
		FROM users
		WHERE id = $1
	`, userID).Scan(&secret, &enabled)
	if err != nil {
		return "", false, err
	}
	if !secret.Valid {
		return "", enabled, nil
	}
	return secret.String, enabled, nil
}

func (d *Database) GetBackupCodes(ctx context.Context, userID int) ([]models.TwoFABackupCode, error) {
	rows, err := d.DB.QueryContext(ctx, `
		SELECT id, user_id, code_hash, used, created_at, used_at
		FROM user_twofa_backup_codes
		WHERE user_id = $1
		ORDER BY id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []models.TwoFABackupCode
	for rows.Next() {
		var code models.TwoFABackupCode
		var usedAt sql.NullTime
		if err := rows.Scan(&code.ID, &code.UserID, &code.CodeHash, &code.Used, &code.CreatedAt, &usedAt); err != nil {
			return nil, err
		}
		if usedAt.Valid {
			code.UsedAt = &usedAt.Time
		}
		codes = append(codes, code)
	}
	return codes, nil
}

func (d *Database) ConsumeBackupCode(ctx context.Context, userID int, candidate string) (bool, error) {
	codes, err := d.GetBackupCodes(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, code := range codes {
		if code.Used {
			continue
		}
		if err := bcrypt.CompareHashAndPassword([]byte(code.CodeHash), []byte(candidate)); err == nil {
			_, err := d.DB.ExecContext(ctx, `
				UPDATE user_twofa_backup_codes
				SET used = true, used_at = CURRENT_TIMESTAMP
				WHERE id = $1
			`, code.ID)
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

func (d *Database) CountRecentTwoFAAttempts(ctx context.Context, userID int, ip string, window time.Duration) (int, error) {
	cutoff := time.Now().Add(-window)
	var count int
	err := d.DB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM twofa_logs
		WHERE user_id = $1
		  AND ip_address = $2
		  AND created_at >= $3
	`, userID, ip, cutoff).Scan(&count)
	return count, err
}

func (d *Database) LogTwoFAAttempt(ctx context.Context, entry *models.TwoFALogEntry) error {
	_, err := d.DB.ExecContext(ctx, `
		INSERT INTO twofa_logs (user_id, ip_address, event, method, success, message)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, entry.UserID, entry.IP, entry.Event, entry.Method, entry.Success, entry.Message)
	return err
}
