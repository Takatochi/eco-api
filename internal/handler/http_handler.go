package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"eco-api/internal/docs"
	"eco-api/internal/model"
	"eco-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBPinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	service service.MeasurementService
	db      DBPinger
}

func New(service service.MeasurementService, db DBPinger) *Handler {
	return &Handler{service: service, db: db}
}

func (h *Handler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "pong"})
}

func (h *Handler) Health(c *gin.Context) {
	ctx, cancel := contextWithTimeout(c, 2*time.Second)
	defer cancel()

	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db not configured"})
		return
	}

	if err := h.db.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db not ready"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) OpenAPI(c *gin.Context) {
	c.Data(http.StatusOK, "application/json; charset=utf-8", docs.OpenAPISpec())
}

func (h *Handler) CreateMeasurement(c *gin.Context) {
	var in model.MeasurementIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	ctx, cancel := contextWithTimeout(c, 3*time.Second)
	defer cancel()

	id, err := h.service.Create(ctx, in)
	if err != nil {
		status := http.StatusBadRequest
		if isUniqueViolation(err) {
			status = http.StatusConflict
		}
		if status == http.StatusBadRequest && !isValidationError(err) {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handler) ListMeasurements(c *gin.Context) {
	ctx, cancel := contextWithTimeout(c, 5*time.Second)
	defer cancel()

	items, err := h.service.List(
		ctx,
		c.Query("deviceId"),
		c.Query("from"),
		c.Query("to"),
		c.Query("limit"),
	)
	if err != nil {
		status := http.StatusBadRequest
		if !isValidationError(err) {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, items)
}

func isValidationError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "required") ||
		strings.Contains(msg, "invalid") ||
		strings.Contains(msg, "must be") ||
		strings.Contains(msg, "range")
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func contextWithTimeout(c *gin.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), timeout)
}
