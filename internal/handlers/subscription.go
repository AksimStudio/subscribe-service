package handlers

import (
	"net/http"
	"time"

	"subscription-service/internal/logger"
	models "subscription-service/internal/model"
	"subscription-service/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type SubscriptionHandler struct {
	repo repository.Repository
	log  *logrus.Logger
}

func NewSubscriptionHandler(repo repository.Repository) *SubscriptionHandler {
	return &SubscriptionHandler{
		repo: repo,
		log:  logger.GetLogger(),
	}
}

func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
	var req models.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.WithError(err).Warn("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, err := time.Parse("01-2006", req.StartDate); err != nil {
		h.log.WithField("start_date", req.StartDate).Warn("Invalid start date format")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start date format, expected MM-YYYY"})
		return
	}

	if req.EndDate != nil {
		if _, err := time.Parse("01-2006", *req.EndDate); err != nil {
			h.log.WithField("end_date", *req.EndDate).Warn("Invalid end date format")
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end date format, expected MM-YYYY"})
			return
		}
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		h.log.WithField("user_id", req.UserID).Warn("Invalid user ID format")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id format"})
		return
	}

	subscription := &models.Subscription{
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      userID,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
	}

	if err := h.repo.Create(subscription); err != nil {
		h.log.WithError(err).Error("Failed to create subscription")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create subscription"})
		return
	}

	h.log.WithField("id", subscription.ID).Info("Subscription created successfully")
	c.JSON(http.StatusCreated, gin.H{
		"id":         subscription.ID,
		"created_at": subscription.CreatedAt,
	})
}

func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.log.WithField("id", c.Param("id")).Warn("Invalid subscription ID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
		return
	}

	subscription, err := h.repo.GetByID(id)
	if err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to get subscription")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get subscription"})
		return
	}

	if subscription == nil {
		h.log.WithField("id", id).Warn("Subscription not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	c.JSON(http.StatusOK, subscription)
}

func (h *SubscriptionHandler) GetAllSubscriptions(c *gin.Context) {
	filter := &models.SubscriptionFilter{
		UserID:      getStringPointer(c.Query("user_id")),
		ServiceName: getStringPointer(c.Query("service_name")),
		StartDate:   getStringPointer(c.Query("start_date")),
		EndDate:     getStringPointer(c.Query("end_date")),
	}

	subscriptions, err := h.repo.GetAll(filter)
	if err != nil {
		h.log.WithError(err).Error("Failed to get subscriptions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get subscriptions"})
		return
	}

	c.JSON(http.StatusOK, subscriptions)
}

func (h *SubscriptionHandler) UpdateSubscription(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.log.WithField("id", c.Param("id")).Warn("Invalid subscription ID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
		return
	}

	var req models.UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.WithError(err).Warn("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.EndDate != nil {
		if _, err := time.Parse("01-2006", *req.EndDate); err != nil {
			h.log.WithField("end_date", *req.EndDate).Warn("Invalid end date format")
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end date format, expected MM-YYYY"})
			return
		}
	}

	if err := h.repo.Update(id, &req); err != nil {
		if err.Error() == "sql: no rows in result set" {
			h.log.WithField("id", id).Warn("Subscription not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		h.log.WithError(err).WithField("id", id).Error("Failed to update subscription")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update subscription"})
		return
	}

	h.log.WithField("id", id).Info("Subscription updated successfully")
	c.JSON(http.StatusOK, gin.H{"message": "subscription updated successfully"})
}

func (h *SubscriptionHandler) DeleteSubscription(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.log.WithField("id", c.Param("id")).Warn("Invalid subscription ID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
		return
	}

	if err := h.repo.Delete(id); err != nil {
		if err.Error() == "sql: no rows in result set" {
			h.log.WithField("id", id).Warn("Subscription not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		h.log.WithError(err).WithField("id", id).Error("Failed to delete subscription")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete subscription"})
		return
	}

	h.log.WithField("id", id).Info("Subscription deleted successfully")
	c.JSON(http.StatusOK, gin.H{"message": "subscription deleted successfully"})
}

func (h *SubscriptionHandler) GetTotalCost(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		h.log.Warn("Missing required parameters")
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_date and end_date are required"})
		return
	}

	// Validate date formats
	if _, err := time.Parse("01-2006", startDate); err != nil {
		h.log.WithField("start_date", startDate).Warn("Invalid start date format")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format, expected MM-YYYY"})
		return
	}

	if _, err := time.Parse("01-2006", endDate); err != nil {
		h.log.WithField("end_date", endDate).Warn("Invalid end date format")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format, expected MM-YYYY"})
		return
	}

	filter := &models.SubscriptionFilter{
		UserID:      getStringPointer(c.Query("user_id")),
		ServiceName: getStringPointer(c.Query("service_name")),
	}

	totalCost, count, err := h.repo.GetTotalCost(filter, startDate, endDate)
	if err != nil {
		h.log.WithError(err).Error("Failed to calculate total cost")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate total cost"})
		return
	}

	response := models.TotalCostResponse{
		TotalCost: totalCost,
		Currency:  "RUB",
		Count:     count,
	}

	h.log.WithFields(logrus.Fields{
		"total_cost": totalCost,
		"count":      count,
	}).Info("Total cost calculated")

	c.JSON(http.StatusOK, response)
}

func (h *SubscriptionHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func getStringPointer(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
