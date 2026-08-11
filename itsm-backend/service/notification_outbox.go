package service

import (
	"context"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/user"
	"itsm-backend/internal/commandbus"
)

type ResourceNotificationCommand struct {
	TenantID         int
	ResourceType     string
	ResourceID       int
	RecipientID      int
	NotificationType string
	Channel          string
	Content          string
	OccurrenceKey    string
}

// EnqueueResourceNotificationTx validates the recipient under the transaction
// tenant and persists one reliable delivery intent with the business mutation.
func EnqueueResourceNotificationTx(ctx context.Context, tx *ent.Tx, request ResourceNotificationCommand) error {
	if tx == nil || request.TenantID <= 0 || request.ResourceID <= 0 || request.RecipientID <= 0 {
		return fmt.Errorf("invalid resource notification command")
	}
	if request.ResourceType == "" || request.NotificationType == "" || request.Channel == "" ||
		request.Content == "" || request.OccurrenceKey == "" {
		return fmt.Errorf("resource notification command fields are required")
	}
	exists, err := tx.User.Query().Where(
		user.IDEQ(request.RecipientID), user.TenantIDEQ(request.TenantID), user.ActiveEQ(true),
	).Exist(ctx)
	if err != nil {
		return fmt.Errorf("validate notification recipient: %w", err)
	}
	if !exists {
		return fmt.Errorf("notification recipient is not active in tenant")
	}
	idempotencyKey := fmt.Sprintf("%s:%s:%d:%s:%d:%s",
		request.OccurrenceKey, request.ResourceType, request.ResourceID,
		request.Channel, request.RecipientID, request.NotificationType)
	_, err = commandbus.EnqueueTx(ctx, tx, commandbus.EnqueueRequest{
		TenantID: request.TenantID, CommandType: commandbus.CommandDeliverNotification,
		AggregateType: request.ResourceType, AggregateID: request.ResourceID,
		IdempotencyKey: idempotencyKey, MaxAttempts: 8,
		Payload: map[string]interface{}{
			"resourceType": request.ResourceType, "resourceId": request.ResourceID,
			"recipientId": request.RecipientID, "channel": request.Channel,
			"type": request.NotificationType, "content": request.Content,
		},
	})
	if err != nil {
		return fmt.Errorf("enqueue resource notification: %w", err)
	}
	return nil
}
