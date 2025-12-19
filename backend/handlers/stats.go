package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"

	"proxy-server/database"
	"proxy-server/models"
)

type StatsHandler struct {
	db *database.Database
}

func NewStatsHandler(db *database.Database) *StatsHandler {
	return &StatsHandler{db: db}
}

func (h *StatsHandler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.db.GetDashboardStats()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch stats")
		return
	}

	respondWithJSON(w, http.StatusOK, stats)
}

func (h *StatsHandler) GetTrafficStats(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 100)

	startDate, err := parseDateParam(r.URL.Query().Get("start_date"), false)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid start_date format")
		return
	}
	endDate, err := parseDateParam(r.URL.Query().Get("end_date"), true)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid end_date format")
		return
	}

	stats, err := h.db.GetTrafficStats(limit, startDate, endDate)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch traffic stats")
		return
	}

	respondWithJSON(w, http.StatusOK, stats)
}

func (h *StatsHandler) GetRequestLogs(w http.ResponseWriter, r *http.Request) {
	filters, err := parseLogFiltersFromRequest(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	logs, err := h.db.GetRequestLogs(filters)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch request logs")
		return
	}

	respondWithJSON(w, http.StatusOK, logs)
}

func (h *StatsHandler) ExportRequestLogs(w http.ResponseWriter, r *http.Request) {
	format := strings.ToLower(r.URL.Query().Get("format"))
	if format != "pdf" && format != "excel" {
		respondWithError(w, http.StatusBadRequest, "Unsupported export format")
		return
	}

	filters, err := parseLogFiltersFromRequest(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if filters.Limit <= 0 || filters.Limit > 5000 {
		filters.Limit = 5000
	}

	logs, err := h.db.GetRequestLogs(filters)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch request logs")
		return
	}

	var data []byte
	var contentType, ext string

	switch format {
	case "pdf":
		data, err = buildLogsPDF(logs)
		contentType = "application/pdf"
		ext = "pdf"
	case "excel":
		data, err = buildLogsExcel(logs)
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		ext = "xlsx"
	}

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to build export")
		return
	}

	filename := fmt.Sprintf("request-logs-%s.%s", time.Now().Format("20060102-150405"), ext)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func parseLogFiltersFromRequest(r *http.Request) (*models.LogFilterOptions, error) {
	q := r.URL.Query()
	filters := &models.LogFilterOptions{
		Method:      q.Get("method"),
		URLContains: q.Get("url"),
		SortBy:      q.Get("sort_by"),
		SortOrder:   q.Get("sort_order"),
		Limit:       parseLimit(q.Get("limit"), 100),
	}

	if offsetStr := q.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filters.Offset = offset
		} else {
			return nil, fmt.Errorf("invalid offset parameter")
		}
	}

	startDate, err := parseDateParam(q.Get("start_date"), false)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date format")
	}
	endDate, err := parseDateParam(q.Get("end_date"), true)
	if err != nil {
		return nil, fmt.Errorf("invalid end_date format")
	}
	filters.StartDate = startDate
	filters.EndDate = endDate

	if userStr := q.Get("user_id"); userStr != "" {
		id, err := strconv.Atoi(userStr)
		if err != nil {
			return nil, fmt.Errorf("invalid user_id")
		}
		filters.UserID = &id
	}

	filters.Username = q.Get("username")

	if statusStr := q.Get("status"); statusStr != "" {
		status, err := strconv.Atoi(statusStr)
		if err != nil {
			return nil, fmt.Errorf("invalid status value")
		}
		filters.StatusCode = &status
	}

	if minSentStr := q.Get("min_sent"); minSentStr != "" {
		value, err := strconv.ParseInt(minSentStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid min_sent value")
		}
		filters.MinBytesSent = &value
	}

	if maxSentStr := q.Get("max_sent"); maxSentStr != "" {
		value, err := strconv.ParseInt(maxSentStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid max_sent value")
		}
		filters.MaxBytesSent = &value
	}

	if minRecvStr := q.Get("min_received"); minRecvStr != "" {
		value, err := strconv.ParseInt(minRecvStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid min_received value")
		}
		filters.MinBytesRecv = &value
	}

	if maxRecvStr := q.Get("max_received"); maxRecvStr != "" {
		value, err := strconv.ParseInt(maxRecvStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid max_received value")
		}
		filters.MaxBytesRecv = &value
	}

	if minDurationStr := q.Get("min_duration"); minDurationStr != "" {
		value, err := strconv.Atoi(minDurationStr)
		if err != nil {
			return nil, fmt.Errorf("invalid min_duration value")
		}
		filters.MinDurationMs = &value
	}

	if maxDurationStr := q.Get("max_duration"); maxDurationStr != "" {
		value, err := strconv.Atoi(maxDurationStr)
		if err != nil {
			return nil, fmt.Errorf("invalid max_duration value")
		}
		filters.MaxDurationMs = &value
	}

	return filters, nil
}

func parseAuditFiltersFromRequest(r *http.Request) (*models.AuditLogFilterOptions, error) {
	q := r.URL.Query()
	filters := &models.AuditLogFilterOptions{
		Username:  q.Get("username"),
		Action:    q.Get("action"),
		Details:   q.Get("details"),
		IPAddress: q.Get("ip_address"),
		SortBy:    q.Get("sort_by"),
		SortOrder: q.Get("sort_order"),
		Limit:     parseLimit(q.Get("limit"), 200),
	}

	startDate, err := parseDateParam(q.Get("start_date"), false)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date format")
	}
	endDate, err := parseDateParam(q.Get("end_date"), true)
	if err != nil {
		return nil, fmt.Errorf("invalid end_date format")
	}
	filters.StartDate = startDate
	filters.EndDate = endDate

	if offsetStr := q.Get("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return nil, fmt.Errorf("invalid offset parameter")
		}
		filters.Offset = offset
	}

	return filters, nil
}

func parseLimit(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	if v, err := strconv.Atoi(value); err == nil && v > 0 {
		return v
	}
	return fallback
}

func parseDateParam(value string, endOfDay bool) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}

	if len(value) == len("2006-01-02") && !strings.Contains(value, "T") {
		t, err := time.Parse("2006-01-02", value)
		if err != nil {
			return nil, err
		}
		if endOfDay {
			t = t.Add(24*time.Hour - time.Nanosecond)
		}
		return &t, nil
	}

	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return &t, nil
	}

	if t, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
		if endOfDay {
			t = t.Add(24*time.Hour - time.Nanosecond)
		}
		return &t, nil
	}

	return nil, fmt.Errorf("invalid date value")
}

func buildLogsPDF(logs []models.RequestLog) ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetFont("Arial", "", 9)
	pdf.AddPage()

	headers := []string{"Time", "User", "Method", "URL", "Status", "Sent (B)", "Received (B)", "Duration (ms)"}
	widths := []float64{40, 30, 20, 120, 20, 25, 30, 30}

	pdf.SetFillColor(240, 240, 240)
	for i, header := range headers {
		pdf.CellFormat(widths[i], 8, header, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	fill := false
	for _, entry := range logs {
		row := []string{
			entry.CreatedAt.Format("2006-01-02 15:04:05"),
			entry.Username,
			entry.Method,
			truncateString(entry.URL, 120),
			fmt.Sprintf("%d", entry.StatusCode),
			fmt.Sprintf("%d", entry.BytesSent),
			fmt.Sprintf("%d", entry.BytesReceived),
			fmt.Sprintf("%d", entry.DurationMs),
		}

		if fill {
			pdf.SetFillColor(250, 250, 250)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		for i, value := range row {
			align := "L"
			if i >= 4 {
				align = "R"
			}
			pdf.CellFormat(widths[i], 6, value, "1", 0, align, true, 0, "")
		}
		pdf.Ln(-1)
		fill = !fill
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildLogsExcel(logs []models.RequestLog) ([]byte, error) {
	file := excelize.NewFile()
	sheetName := file.GetSheetName(0)

	headers := []string{"Time", "User", "Method", "URL", "Status", "Sent (B)", "Received (B)", "Duration (ms)"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		file.SetCellValue(sheetName, cell, header)
	}

	for idx, entry := range logs {
		row := idx + 2
		values := []interface{}{
			entry.CreatedAt.Format("2006-01-02 15:04:05"),
			entry.Username,
			entry.Method,
			entry.URL,
			entry.StatusCode,
			entry.BytesSent,
			entry.BytesReceived,
			entry.DurationMs,
		}
		for col, value := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			file.SetCellValue(sheetName, cell, value)
		}
	}

	if idx, err := file.GetSheetIndex(sheetName); err == nil {
		file.SetActiveSheet(idx)
	}
	file.AutoFilter(sheetName, "A1:H1", []excelize.AutoFilterOptions{})
	file.SetColWidth(sheetName, "A", "A", 22)
	file.SetColWidth(sheetName, "B", "B", 18)
	file.SetColWidth(sheetName, "C", "C", 12)
	file.SetColWidth(sheetName, "D", "D", 60)

	var buf bytes.Buffer
	if err := file.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func truncateString(value string, limit int) string {
	if len([]rune(value)) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit-3]) + "..."
}

func (h *StatsHandler) GetLogRetention(w http.ResponseWriter, r *http.Request) {
	days, err := h.db.GetLogRetentionDays()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch retention settings")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]int{"retention_days": days})
}

type clearLogsRequest struct {
	Days int `json:"days"`
}

func (h *StatsHandler) ClearRequestLogs(w http.ResponseWriter, r *http.Request) {
	var req clearLogsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Days <= 0 {
		respondWithError(w, http.StatusBadRequest, "Retention days must be greater than zero")
		return
	}

	if err := h.db.SetLogRetentionDays(req.Days); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save retention setting")
		return
	}

	deleted, err := h.db.DeleteRequestLogsOlderThan(req.Days)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to clear logs")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"retention_days": req.Days,
		"deleted_count":  deleted,
		"message":        fmt.Sprintf("Removed %d log entries older than %d days", deleted, req.Days),
	})
}

func (h *StatsHandler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	filters, err := parseAuditFiltersFromRequest(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	logs, err := h.db.GetAuditLogs(filters)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch audit logs")
		return
	}

	respondWithJSON(w, http.StatusOK, logs)
}
