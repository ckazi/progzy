package models

import "time"

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Email        string    `json:"email"`
	Comment      string    `json:"comment"`
	IsAdmin      bool      `json:"is_admin"`
	IsActive     bool      `json:"is_active"`
	ProxyType    string    `json:"proxy_type"`
	TwoFAEnabled bool      `json:"twofa_enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Whitelist    []string  `json:"whitelist,omitempty"`
	Blacklist    []string  `json:"blacklist,omitempty"`
}

type UserCreate struct {
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Email     string   `json:"email"`
	Comment   string   `json:"comment"`
	IsAdmin   bool     `json:"is_admin"`
	ProxyType string   `json:"proxy_type"`
	Whitelist []string `json:"whitelist"`
	Blacklist []string `json:"blacklist"`
}

type UserUpdate struct {
	Email     *string   `json:"email"`
	Comment   *string   `json:"comment"`
	IsAdmin   *bool     `json:"is_admin"`
	IsActive  *bool     `json:"is_active"`
	Password  *string   `json:"password,omitempty"`
	ProxyType *string   `json:"proxy_type,omitempty"`
	Whitelist *[]string `json:"whitelist,omitempty"`
	Blacklist *[]string `json:"blacklist,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token       string `json:"token"`
	User        User   `json:"user"`
	Message     string `json:"message,omitempty"`
	Requires2FA bool   `json:"requires_2fa,omitempty"`
	TempToken   string `json:"temp_token,omitempty"`
}

type ProxySetting struct {
	ID          int       `json:"id"`
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TrafficStats struct {
	ID            int       `json:"id"`
	UserID        int       `json:"user_id"`
	Username      string    `json:"username,omitempty"`
	BytesSent     int64     `json:"bytes_sent"`
	BytesReceived int64     `json:"bytes_received"`
	RequestCount  int       `json:"request_count"`
	Date          time.Time `json:"date"`
}

type RequestLog struct {
	ID            int       `json:"id"`
	UserID        *int      `json:"user_id"`
	Username      string    `json:"username,omitempty"`
	Method        string    `json:"method"`
	URL           string    `json:"url"`
	StatusCode    int       `json:"status_code"`
	BytesSent     int64     `json:"bytes_sent"`
	BytesReceived int64     `json:"bytes_received"`
	DurationMs    int       `json:"duration_ms"`
	CreatedAt     time.Time `json:"created_at"`
}

type UserProxySettings struct {
	ProxyType string   `json:"proxy_type"`
	Whitelist []string `json:"whitelist"`
	Blacklist []string `json:"blacklist"`
}

type AdminAuditLog struct {
	ID        int       `json:"id"`
	UserID    *int      `json:"user_id"`
	Username  string    `json:"username,omitempty"`
	Action    string    `json:"action"`
	Details   string    `json:"details,omitempty"`
	IPAddress string    `json:"ip_address,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type LogFilterOptions struct {
	StartDate     *time.Time
	EndDate       *time.Time
	UserID        *int
	Username      string
	Method        string
	URLContains   string
	StatusCode    *int
	MinBytesSent  *int64
	MaxBytesSent  *int64
	MinBytesRecv  *int64
	MaxBytesRecv  *int64
	MinDurationMs *int
	MaxDurationMs *int
	SortBy        string
	SortOrder     string
	Limit         int
	Offset        int
}

type StatsResponse struct {
	TotalUsers     int            `json:"total_users"`
	ActiveUsers    int            `json:"active_users"`
	TotalRequests  int64          `json:"total_requests"`
	TotalBytesSent int64          `json:"total_bytes_sent"`
	TotalBytesRecv int64          `json:"total_bytes_received"`
	UserStats      []TrafficStats `json:"user_stats,omitempty"`
	RecentLogs     []RequestLog   `json:"recent_logs,omitempty"`
}

type AuditLogFilterOptions struct {
	StartDate *time.Time
	EndDate   *time.Time
	Username  string
	Action    string
	Details   string
	IPAddress string
	SortBy    string
	SortOrder string
	Limit     int
	Offset    int
}

type InitSetupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type TwoFABackupCode struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	CodeHash  string     `json:"-"`
	Used      bool       `json:"used"`
	CreatedAt time.Time  `json:"created_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
}

type TwoFASetupResponse struct {
	Secret     string `json:"secret"`
	OTPAuthURL string `json:"otpauth_url"`
	QRCode     string `json:"qr_code"`
}

type TwoFAVerifyRequest struct {
	Code       string `json:"code"`
	BackupCode string `json:"backup_code"`
	TempToken  string `json:"temp_token,omitempty"`
}

type TwoFAVerifyResponse struct {
	Token       string   `json:"token,omitempty"`
	User        *User    `json:"user,omitempty"`
	Message     string   `json:"message,omitempty"`
	BackupCodes []string `json:"backup_codes,omitempty"`
}

type TwoFABackupCodesResponse struct {
	Codes []string `json:"codes"`
}

type TwoFALogEntry struct {
	UserID    int
	IP        string
	Method    string
	Event     string
	Success   bool
	Message   string
	CreatedAt time.Time
}
