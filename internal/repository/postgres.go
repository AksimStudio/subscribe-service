package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"subscription-service/internal/logger"
	models "subscription-service/internal/model"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Repository interface {
	Create(sub *models.Subscription) error
	GetByID(id uuid.UUID) (*models.Subscription, error)
	GetAll(filter *models.SubscriptionFilter) ([]*models.Subscription, error)
	Update(id uuid.UUID, req *models.UpdateSubscriptionRequest) error
	Delete(id uuid.UUID) error
	GetTotalCost(filter *models.SubscriptionFilter, startDate, endDate string) (int, int, error)
}

type PostgresRepository struct {
	db  *sql.DB
	log *logrus.Logger
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{
		db:  db,
		log: logger.GetLogger(),
	}
}

func (r *PostgresRepository) Create(sub *models.Subscription) error {
	query := `
        INSERT INTO subscriptions (id, service_name, price, user_id, start_date, end_date, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `

	sub.ID = uuid.New()
	sub.CreatedAt = time.Now()
	sub.UpdatedAt = time.Now()

	r.log.WithFields(logrus.Fields{
		"id":           sub.ID,
		"user_id":      sub.UserID,
		"service_name": sub.ServiceName,
	}).Info("Creating new subscription")

	_, err := r.db.Exec(query,
		sub.ID,
		sub.ServiceName,
		sub.Price,
		sub.UserID,
		sub.StartDate,
		sub.EndDate,
		sub.CreatedAt,
		sub.UpdatedAt,
	)

	if err != nil {
		r.log.WithError(err).Error("Failed to create subscription")
		return err
	}

	return nil
}

func (r *PostgresRepository) GetByID(id uuid.UUID) (*models.Subscription, error) {
	query := `
        SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at
        FROM subscriptions
        WHERE id = $1
    `

	r.log.WithField("id", id).Info("Fetching subscription by ID")

	sub := &models.Subscription{}
	err := r.db.QueryRow(query, id).Scan(
		&sub.ID,
		&sub.ServiceName,
		&sub.Price,
		&sub.UserID,
		&sub.StartDate,
		&sub.EndDate,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			r.log.WithField("id", id).Warn("Subscription not found")
			return nil, nil
		}
		r.log.WithError(err).WithField("id", id).Error("Failed to fetch subscription")
		return nil, err
	}

	return sub, nil
}

func (r *PostgresRepository) GetAll(filter *models.SubscriptionFilter) ([]*models.Subscription, error) {
	query := `
        SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at
        FROM subscriptions
        WHERE 1=1
    `
	args := []interface{}{}
	conditions := []string{}

	if filter.UserID != nil && *filter.UserID != "" {
		args = append(args, *filter.UserID)
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", len(args)))
	}

	if filter.ServiceName != nil && *filter.ServiceName != "" {
		args = append(args, "%"+*filter.ServiceName+"%")
		conditions = append(conditions, fmt.Sprintf("service_name ILIKE $%d", len(args)))
	}

	if filter.StartDate != nil && *filter.StartDate != "" {
		args = append(args, *filter.StartDate)
		conditions = append(conditions, fmt.Sprintf("start_date >= $%d", len(args)))
	}

	if filter.EndDate != nil && *filter.EndDate != "" {
		args = append(args, *filter.EndDate)
		conditions = append(conditions, fmt.Sprintf("(end_date <= $%d OR end_date IS NULL)", len(args)))
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

	r.log.WithField("filter", filter).Info("Fetching all subscriptions")

	rows, err := r.db.Query(query, args...)
	if err != nil {
		r.log.WithError(err).Error("Failed to fetch subscriptions")
		return nil, err
	}
	defer rows.Close()

	var subscriptions []*models.Subscription
	for rows.Next() {
		sub := &models.Subscription{}
		err := rows.Scan(
			&sub.ID,
			&sub.ServiceName,
			&sub.Price,
			&sub.UserID,
			&sub.StartDate,
			&sub.EndDate,
			&sub.CreatedAt,
			&sub.UpdatedAt,
		)
		if err != nil {
			r.log.WithError(err).Error("Failed to scan subscription")
			return nil, err
		}
		subscriptions = append(subscriptions, sub)
	}

	r.log.WithField("count", len(subscriptions)).Info("Subscriptions fetched successfully")
	return subscriptions, nil
}

func (r *PostgresRepository) Update(id uuid.UUID, req *models.UpdateSubscriptionRequest) error {
	query := "UPDATE subscriptions SET updated_at = $1"
	args := []interface{}{time.Now()}
	updates := []string{}

	if req.ServiceName != nil {
		args = append(args, *req.ServiceName)
		updates = append(updates, fmt.Sprintf("service_name = $%d", len(args)))
	}

	if req.Price != nil {
		args = append(args, *req.Price)
		updates = append(updates, fmt.Sprintf("price = $%d", len(args)))
	}

	if req.EndDate != nil {
		args = append(args, *req.EndDate)
		updates = append(updates, fmt.Sprintf("end_date = $%d", len(args)))
	}

	if len(updates) == 0 {
		return nil
	}

	query += ", " + strings.Join(updates, ", ")
	args = append(args, id)
	query += fmt.Sprintf(" WHERE id = $%d", len(args))

	r.log.WithField("id", id).Info("Updating subscription")

	result, err := r.db.Exec(query, args...)
	if err != nil {
		r.log.WithError(err).WithField("id", id).Error("Failed to update subscription")
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		r.log.WithField("id", id).Warn("Subscription not found for update")
		return sql.ErrNoRows
	}

	return nil
}

func (r *PostgresRepository) Delete(id uuid.UUID) error {
	query := "DELETE FROM subscriptions WHERE id = $1"

	r.log.WithField("id", id).Info("Deleting subscription")

	result, err := r.db.Exec(query, id)
	if err != nil {
		r.log.WithError(err).WithField("id", id).Error("Failed to delete subscription")
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		r.log.WithField("id", id).Warn("Subscription not found for deletion")
		return sql.ErrNoRows
	}

	return nil
}

func (r *PostgresRepository) GetTotalCost(filter *models.SubscriptionFilter, startDate, endDate string) (int, int, error) {
	query := `
        SELECT COALESCE(SUM(price), 0), COUNT(*)
        FROM subscriptions
        WHERE start_date <= $1 AND (end_date IS NULL OR end_date >= $2)
    `
	args := []interface{}{endDate, startDate}
	conditions := []string{}

	if filter.UserID != nil && *filter.UserID != "" {
		args = append(args, *filter.UserID)
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", len(args)))
	}

	if filter.ServiceName != nil && *filter.ServiceName != "" {
		args = append(args, *filter.ServiceName)
		conditions = append(conditions, fmt.Sprintf("service_name = $%d", len(args)))
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	r.log.WithFields(logrus.Fields{
		"start_date": startDate,
		"end_date":   endDate,
		"filter":     filter,
	}).Info("Calculating total cost")

	var totalCost, count int
	err := r.db.QueryRow(query, args...).Scan(&totalCost, &count)
	if err != nil {
		r.log.WithError(err).Error("Failed to calculate total cost")
		return 0, 0, err
	}

	r.log.WithFields(logrus.Fields{
		"total_cost": totalCost,
		"count":      count,
	}).Info("Total cost calculated successfully")

	return totalCost, count, nil
}
