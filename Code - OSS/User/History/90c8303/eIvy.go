package service

import (
	"github.com/skillissu3e/notify-platform/notification-service/internal/repository"
)

type AuditService struct {
	repo *repository.AuditLogRepository
}

func NewAuditService(repo *repository.AuditLogRepository) *AuditService {
	return &AuditService{repo: repo}
}

func (s *AuditService) LogNotificationSent(userID int, notificationID string, channel string, status string) error {
	return s.repo.Create(&repository.AuditLog{
		UserID:     userID,
		Action:     "NOTIFICATION_SENT",
		EntityType: "notification",
		EntityID:   notificationID,
		Status:     status,
		Metadata: map[string]interface{}{
			"channel": channel,
		},
	})
}

func (s *AuditService) LogUserAction(userID int, action string, entityType string, entityID string, status string) error {
	return s.repo.Create(&repository.AuditLog{
		UserID:     userID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Status:     status,
	})
}

func (s *AuditService) GetUserHistory(userID int, limit int) ([]repository.AuditLog, error) {
	return s.repo.FindByUser(userID, limit)
}
