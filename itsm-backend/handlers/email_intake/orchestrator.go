package email_intake

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"itsm-backend/connector"
	"itsm-backend/ent"
	"itsm-backend/ent/emailconversation"
	"itsm-backend/ent/emailintakeanalysis"
	"itsm-backend/ent/inboundemailmessage"
	"itsm-backend/ent/sourceorganization"
	"itsm-backend/ent/systemconfig"
	"itsm-backend/internal/commandbus"
	"itsm-backend/service"
)

type IntakeMode string

const (
	ModeObserveOnly   IntakeMode = "observeOnly"
	ModeManualConfirm IntakeMode = "manualConfirm"
	ModeAutoCreate    IntakeMode = "autoCreate"
)

type OrchestratorConfig struct {
	Mode                     IntakeMode
	ConfidenceCutoff         float64
	AutomationReporterUserID int
	DefaultAssignmentGroupID *int
}

type ReceivedEmail struct {
	Provider            string
	MailboxInstanceKey  string
	UIDValidity         uint64
	UID                 uint64
	ExternalMessageID   string
	InReplyTo           string
	References          []string
	FromAddress         string
	ToAddresses         []string
	ReplyToAddress      string
	Subject             string
	PlainText           string
	HTMLBody            string
	RawMIME             []byte
	ReceivedAt          time.Time
	SenderAuthenticated bool
}

type EmailIntakeOrchestrator struct {
	client          *ent.Client
	extractor       *EmailIntakeExtractor
	resolver        *Resolver
	incidentService *service.IncidentService
	onCall          *OnCallService
	config          OrchestratorConfig
}

func NewEmailIntakeOrchestrator(client *ent.Client, extractor *EmailIntakeExtractor, incidentService *service.IncidentService, config OrchestratorConfig) *EmailIntakeOrchestrator {
	if config.Mode == "" {
		config.Mode = ModeObserveOnly
	}
	if config.ConfidenceCutoff <= 0 {
		config.ConfidenceCutoff = DefaultConfidenceCutoff
	}
	return &EmailIntakeOrchestrator{client: client, extractor: extractor, resolver: NewResolver(client), incidentService: incidentService, onCall: NewOnCallService(client), config: config}
}

func (o *EmailIntakeOrchestrator) IngestConnectorMessage(ctx context.Context, tenantID int, instanceKey string, message *connector.InboundMessage) error {
	if message == nil {
		return errors.New("inbound connector message is required")
	}
	uid, _ := numericUint64(message.Extras["uid"])
	uidValidity, _ := numericUint64(message.Extras["uidValidity"])
	toAddresses, _ := message.Extras["toAddresses"].([]string)
	references, _ := message.Extras["references"].([]string)
	_, err := o.Ingest(ctx, tenantID, ReceivedEmail{
		Provider: "imap", MailboxInstanceKey: instanceKey, UIDValidity: uidValidity, UID: uid,
		ExternalMessageID: stringExtra(message.Extras, "externalMessageId"), InReplyTo: stringExtra(message.Extras, "inReplyTo"), References: references,
		FromAddress: message.UserID, ToAddresses: toAddresses, ReplyToAddress: stringExtra(message.Extras, "replyToAddress"), Subject: stringExtra(message.Extras, "subject"),
		PlainText: message.Content, HTMLBody: stringExtra(message.Extras, "htmlBody"), RawMIME: []byte(message.Raw), ReceivedAt: message.ReceivedAt,
	})
	return err
}

func (o *EmailIntakeOrchestrator) Ingest(ctx context.Context, tenantID int, email ReceivedEmail) (*ent.EmailConversation, error) {
	if tenantID <= 0 || email.MailboxInstanceKey == "" || email.FromAddress == "" || email.UID == 0 {
		return nil, errors.New("tenant, mailbox, sender and IMAP UID are required")
	}
	if len(email.RawMIME) > 25*1024*1024 {
		return nil, errors.New("email exceeds 25 MiB limit")
	}
	if email.ReceivedAt.IsZero() {
		email.ReceivedAt = time.Now()
	}

	// Phase 1 + 2 in a single transaction to ensure idempotent write:
	// two concurrent poll goroutines hitting the same UID must not create duplicates.
	var conversation *ent.EmailConversation
	var message *ent.InboundEmailMessage

	tx, err := o.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("start idempotent-write transaction: %w", err)
	}
	rollback := func(cause error) error { _ = tx.Rollback(); return cause }

	// Idempotency check inside the transaction
	existing, err := tx.InboundEmailMessage.Query().Where(
		inboundemailmessage.TenantIDEQ(tenantID),
		inboundemailmessage.MailboxInstanceKeyEQ(email.MailboxInstanceKey),
		inboundemailmessage.UIDValidityEQ(email.UIDValidity),
		inboundemailmessage.UIDEQ(email.UID),
	).WithConversation().Only(ctx)
	if err == nil {
		_ = tx.Rollback()
		if existing.ProcessingStatus != "PARSED" {
			if enqueueErr := o.enqueueMessageProcessing(ctx, tenantID, existing.ID); enqueueErr != nil {
				return nil, enqueueErr
			}
		}
		return existing.Edges.Conversation, nil
	}
	if !ent.IsNotFound(err) {
		return nil, fmt.Errorf("check inbound email idempotency: %w", err)
	}

	conversation, err = o.findOrCreateConversationTx(ctx, tx, tenantID, email)
	if err != nil {
		return nil, rollback(err)
	}

	hash := sha256.Sum256(email.RawMIME)
	sanitizedHTML := bluemonday.StrictPolicy().Sanitize(email.HTMLBody)
	message, err = tx.InboundEmailMessage.Create().
		SetTenantID(tenantID).SetConversationID(conversation.ID).
		SetProvider(defaultString(email.Provider, "imap")).SetMailboxInstanceKey(email.MailboxInstanceKey).
		SetUIDValidity(email.UIDValidity).SetUID(email.UID).SetExternalMessageID(email.ExternalMessageID).
		SetInReplyTo(email.InReplyTo).SetReferences(email.References).SetFromAddress(email.FromAddress).
		SetToAddresses(email.ToAddresses).SetReplyToAddress(email.ReplyToAddress).SetSubject(email.Subject).
		SetPlainText(limitRunes(email.PlainText, 20000)).SetSanitizedHTML(limitRunes(sanitizedHTML, 50000)).
		SetRawMime(email.RawMIME).SetRawSha256(hex.EncodeToString(hash[:])).SetReceivedAt(email.ReceivedAt).Save(ctx)

	if ent.IsConstraintError(err) {
		// Another goroutine won the race — fetch the winner and abort this write.
		_ = tx.Rollback()
		existing, reErr := o.client.InboundEmailMessage.Query().Where(
			inboundemailmessage.TenantIDEQ(tenantID),
			inboundemailmessage.MailboxInstanceKeyEQ(email.MailboxInstanceKey),
			inboundemailmessage.UIDValidityEQ(email.UIDValidity),
			inboundemailmessage.UIDEQ(email.UID),
		).WithConversation().Only(ctx)
		if reErr != nil {
			return nil, fmt.Errorf("idempotency conflict and re-fetch failed: %w", reErr)
		}
		if existing.ProcessingStatus != "PARSED" {
			if enqueueErr := o.enqueueMessageProcessing(ctx, tenantID, existing.ID); enqueueErr != nil {
				return nil, enqueueErr
			}
		}
		return existing.Edges.Conversation, nil
	}
	if err != nil {
		return nil, rollback(fmt.Errorf("persist inbound email: %w", err))
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit idempotent-write transaction: %w", err)
	}

	return o.process(ctx, tenantID, conversation, message, email.SenderAuthenticated, true)
}

func (o *EmailIntakeOrchestrator) process(ctx context.Context, tenantID int, conversation *ent.EmailConversation, message *ent.InboundEmailMessage, senderAuthenticated, enqueueRetry bool) (*ent.EmailConversation, error) {
	started := time.Now()
	result, raw, extractionErr := o.extractor.Extract(ctx, message.Subject, message.PlainText)
	if extractionErr != nil {
		tx, txErr := o.client.Tx(ctx)
		if txErr != nil {
			return nil, fmt.Errorf("start failed-analysis transaction: %w", txErr)
		}
		rollback := func(cause error) (*ent.EmailConversation, error) { _ = tx.Rollback(); return nil, cause }
		if _, txErr = tx.EmailIntakeAnalysis.Create().SetTenantID(tenantID).SetConversationID(conversation.ID).SetMessageID(message.ID).SetPromptVersion(EmailIntakePromptVersion).SetRawResult(raw).SetLatencyMs(time.Since(started).Milliseconds()).SetProvider("llm_gateway").SetModel(o.extractor.model).SetStatus("failed").SetValidationError(extractionErr.Error()).Save(ctx); txErr != nil {
			return rollback(fmt.Errorf("persist failed email intake analysis: %w", txErr))
		}
		if _, txErr = tx.InboundEmailMessage.UpdateOneID(message.ID).SetProcessingStatus("RETRYABLE_FAILED").SetLastError(extractionErr.Error()).Save(ctx); txErr != nil {
			return rollback(fmt.Errorf("persist retryable email status: %w", txErr))
		}
		if txErr = tx.Commit(); txErr != nil {
			return nil, fmt.Errorf("commit failed-analysis transaction: %w", txErr)
		}
		updated, updateErr := o.updateConversation(ctx, conversation, ResolutionManualReview, IntakeFields{}, Resolution{Status: ResolutionManualReview, Reasons: []string{"ai_extraction_failed"}})
		if enqueueRetry {
			if enqueueErr := o.enqueueMessageProcessing(ctx, tenantID, message.ID); enqueueErr != nil {
				return nil, enqueueErr
			}
		}
		if updateErr != nil {
			return nil, updateErr
		}
		if enqueueRetry {
			return updated, nil
		}
		return updated, extractionErr
	}
	fields := result.IntakeFields()
	if existingFields, mapErr := fieldsFromMap(conversation.CanonicalData); mapErr == nil {
		fields = mergeIntakeFields(existingFields, fields, conversation.FieldSources)
	}
	if sourceName, sourceID := o.sourceOrganizationForSender(ctx, tenantID, message.FromAddress); sourceID != 0 {
		fields.SourceOrganizationName = sourceName
	}
	resolution, err := o.resolver.Resolve(ctx, tenantID, fields)
	if err != nil {
		return nil, err
	}
	if fields.Confidence < o.config.ConfidenceCutoff {
		resolution.Status = ResolutionManualReview
		resolution.Reasons = append(resolution.Reasons, "low_ai_confidence")
	}
	// Standard IMAP does not provide a trustworthy sender-authentication
	// assertion. Keep the result reviewable, but never auto-authorize it.
	if !senderAuthenticated {
		resolution.Status = ResolutionManualReview
		resolution.Reasons = append(resolution.Reasons, "sender_identity_unverified")
	}
	resultMap := map[string]interface{}{}
	encoded, _ := json.Marshal(result)
	_ = json.Unmarshal(encoded, &resultMap)
	tx, err := o.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("start accepted-analysis transaction: %w", err)
	}
	if _, err = tx.EmailIntakeAnalysis.Create().SetTenantID(tenantID).SetConversationID(conversation.ID).SetMessageID(message.ID).SetPromptVersion(EmailIntakePromptVersion).SetRawResult(raw).SetLatencyMs(time.Since(started).Milliseconds()).SetProvider("llm_gateway").SetModel(o.extractor.model).SetResult(resultMap).SetConfidence(result.Confidence).SetStatus("accepted").Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("persist email intake analysis: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit accepted-analysis transaction: %w", err)
	}
	updated, err := o.updateConversation(ctx, conversation, resolution.Status, fields, resolution)
	if err != nil {
		if enqueueErr := o.enqueueMessageProcessing(ctx, tenantID, message.ID); enqueueErr != nil {
			return nil, fmt.Errorf("update conversation: %v; schedule recovery: %w", err, enqueueErr)
		}
		return nil, err
	}
	if _, err = o.client.InboundEmailMessage.UpdateOneID(message.ID).Where(inboundemailmessage.TenantIDEQ(tenantID)).SetProcessingStatus("PARSED").ClearLastError().Save(ctx); err != nil {
		return nil, fmt.Errorf("persist parsed email status: %w", err)
	}
	if resolution.Status == ResolutionNeedInfo || resolution.Status == ResolutionAmbiguous {
		if err := o.enqueueMissingInformationReply(ctx, tenantID, updated, message, resolution.MissingFields); err != nil {
			return nil, err
		}
	}
	if resolution.Status == ResolutionVerified && o.runtimeConfig(ctx, tenantID).Mode == ModeAutoCreate {
		return o.Confirm(ctx, tenantID, updated.ID, updated.Version)
	}
	return updated, nil
}

func (o *EmailIntakeOrchestrator) Confirm(ctx context.Context, tenantID, conversationID, version int) (*ent.EmailConversation, error) {
	conversation, err := o.client.EmailConversation.Query().Where(emailconversation.IDEQ(conversationID), emailconversation.TenantIDEQ(tenantID)).WithMessages(func(q *ent.InboundEmailMessageQuery) { q.Order(ent.Desc(inboundemailmessage.FieldReceivedAt)).Limit(1) }).Only(ctx)
	if err != nil {
		return nil, err
	}
	if conversation.Version != version {
		return nil, errors.New("conversation was modified by another operator")
	}
	if conversation.Status == "INCIDENT_CREATED" {
		return conversation, nil
	}
	runtimeConfig := o.runtimeConfig(ctx, tenantID)
	fields, err := fieldsFromMap(conversation.CanonicalData)
	if err != nil {
		return nil, err
	}
	resolution, err := o.resolver.Resolve(ctx, tenantID, fields)
	if err != nil {
		return nil, err
	}
	if resolution.Status != ResolutionVerified {
		return nil, fmt.Errorf("conversation is not eligible for incident creation: %s", resolution.Status)
	}
	if o.incidentService == nil || runtimeConfig.AutomationReporterUserID <= 0 {
		return nil, errors.New("incident automation reporter is not configured")
	}
	var assigneeID *int
	if runtimeConfig.DefaultAssignmentGroupID != nil {
		current, currentErr := o.onCall.CurrentResolver(ctx, tenantID, *runtimeConfig.DefaultAssignmentGroupID, time.Now())
		if currentErr != nil && !errors.Is(currentErr, ErrNoOnCall) {
			return nil, currentErr
		}
		if current != nil {
			assigneeID = &current.UserID
		}
	}
	incidentResponse, err := o.incidentService.CreateFromEmail(ctx, tenantID, service.EmailIncidentCommand{ConversationID: conversation.ID, SupportContractID: resolution.SupportContractID, ReporterUserID: runtimeConfig.AutomationReporterUserID, AssignmentGroupID: runtimeConfig.DefaultAssignmentGroupID, AssigneeID: assigneeID, Title: defaultString(fields.Title, "邮件报障"), Description: fields.Description, Impact: defaultString(fields.Impact, "medium"), Urgency: defaultString(fields.Urgency, "medium"), Category: "network", Metadata: map[string]interface{}{"customerId": resolution.CustomerID, "branchId": resolution.BranchID, "sourceOrganizationId": resolution.SourceOrganizationID, "aiConfidence": fields.Confidence}})
	if err != nil {
		return nil, err
	}
	updated, err := o.client.EmailConversation.UpdateOneID(conversation.ID).Where(emailconversation.TenantIDEQ(tenantID), emailconversation.VersionEQ(version)).SetStatus("INCIDENT_CREATED").SetCustomerID(resolution.CustomerID).SetNillableBranchID(optionalInt(resolution.BranchID)).SetSupportContractID(resolution.SupportContractID).AddVersion(1).Save(ctx)
	if err != nil {
		return nil, err
	}
	if len(conversation.Edges.Messages) > 0 {
		if err := o.enqueueIncidentCreatedReply(ctx, tenantID, updated, conversation.Edges.Messages[0], incidentResponse.IncidentNumber); err != nil {
			return nil, err
		}
	}
	return updated, nil
}

// Retry reruns extraction for the latest retryable message. It intentionally
// keeps senderAuthenticated=false because a manual API action cannot upgrade
// an untrusted RFC From header into an authenticated identity.
func (o *EmailIntakeOrchestrator) Retry(ctx context.Context, tenantID, conversationID, version int) (*ent.EmailConversation, error) {
	conversation, err := o.client.EmailConversation.Query().Where(
		emailconversation.IDEQ(conversationID), emailconversation.TenantIDEQ(tenantID), emailconversation.VersionEQ(version),
	).WithMessages(func(q *ent.InboundEmailMessageQuery) {
		q.Where(inboundemailmessage.ProcessingStatusEQ("RETRYABLE_FAILED")).Order(ent.Desc(inboundemailmessage.FieldReceivedAt)).Limit(1)
	}).Only(ctx)
	if err != nil {
		return nil, err
	}
	if len(conversation.Edges.Messages) == 0 {
		return nil, errors.New("no retryable inbound email is available")
	}
	message := conversation.Edges.Messages[0]
	return o.process(ctx, tenantID, conversation, message, false, false)
}

func (o *EmailIntakeOrchestrator) RetryMessage(ctx context.Context, tenantID, messageID int) error {
	message, err := o.client.InboundEmailMessage.Query().Where(
		inboundemailmessage.IDEQ(messageID), inboundemailmessage.TenantIDEQ(tenantID),
	).WithConversation().Only(ctx)
	if err != nil {
		return err
	}
	if message.ProcessingStatus == "PARSED" {
		return nil
	}
	if (message.ProcessingStatus != "RETRYABLE_FAILED" && message.ProcessingStatus != "RECEIVED") || message.Edges.Conversation == nil {
		return errors.New("inbound email is not retryable")
	}
	_, err = o.process(ctx, tenantID, message.Edges.Conversation, message, false, false)
	return err
}

func (o *EmailIntakeOrchestrator) enqueueMessageProcessing(ctx context.Context, tenantID, messageID int) error {
	_, err := commandbus.Enqueue(ctx, o.client, commandbus.EnqueueRequest{
		TenantID: tenantID, CommandType: commandbus.CommandProcessIntakeEmail,
		AggregateType: "inbound_email_message", AggregateID: messageID,
		IdempotencyKey: fmt.Sprintf("email-intake-process:%d:%d", tenantID, messageID),
		Payload:        map[string]interface{}{"messageId": messageID},
	})
	if err != nil && !ent.IsConstraintError(err) {
		return fmt.Errorf("enqueue email intake processing: %w", err)
	}
	return nil
}

func (o *EmailIntakeOrchestrator) Revalidate(ctx context.Context, tenantID, conversationID, version int) (*ent.EmailConversation, error) {
	conversation, err := o.client.EmailConversation.Query().Where(emailconversation.IDEQ(conversationID), emailconversation.TenantIDEQ(tenantID)).Only(ctx)
	if err != nil {
		return nil, err
	}
	if conversation.Version != version {
		return nil, errors.New("conversation was modified by another operator")
	}
	fields, err := fieldsFromMap(conversation.CanonicalData)
	if err != nil {
		return nil, err
	}
	resolution, err := o.resolver.Resolve(ctx, tenantID, fields)
	if err != nil {
		return nil, err
	}
	if fields.Confidence < o.config.ConfidenceCutoff {
		resolution.Status = ResolutionManualReview
		resolution.Reasons = append(resolution.Reasons, "low_ai_confidence")
	}
	return o.updateConversation(ctx, conversation, resolution.Status, fields, resolution)
}

func (o *EmailIntakeOrchestrator) ApplyCorrections(ctx context.Context, tenantID, conversationID, version, reviewerID int, fields IntakeFields) (*ent.EmailConversation, error) {
	conversation, err := o.client.EmailConversation.Query().Where(emailconversation.IDEQ(conversationID), emailconversation.TenantIDEQ(tenantID)).WithAnalyses().Only(ctx)
	if err != nil {
		return nil, err
	}
	if conversation.Version != version {
		return nil, errors.New("conversation was modified by another operator")
	}
	resolution, err := o.resolver.Resolve(ctx, tenantID, fields)
	if err != nil {
		return nil, err
	}
	updated, err := o.updateConversation(ctx, conversation, resolution.Status, fields, resolution)
	if err != nil {
		return nil, err
	}
	corrections, _ := fieldsToMap(fields)
	fieldSources := copyMap(conversation.FieldSources)
	for _, key := range []string{"sourceOrganizationName", "customerName", "branchName", "reportedContractNumber", "title", "description", "occurredAt", "impact", "urgency"} {
		fieldSources[key] = "human"
	}
	if _, err := o.client.EmailConversation.UpdateOneID(updated.ID).Where(emailconversation.TenantIDEQ(tenantID)).SetFieldSources(fieldSources).Save(ctx); err != nil {
		return nil, fmt.Errorf("persist correction field sources: %w", err)
	}
	if len(conversation.Edges.Analyses) > 0 {
		latest := conversation.Edges.Analyses[len(conversation.Edges.Analyses)-1]
		if _, err := o.client.EmailIntakeAnalysis.UpdateOneID(latest.ID).Where(emailintakeanalysis.TenantIDEQ(tenantID)).SetStatus("corrected").SetCorrections(corrections).SetReviewedBy(reviewerID).Save(ctx); err != nil {
			return nil, fmt.Errorf("persist corrected analysis: %w", err)
		}
	}
	return updated, nil
}

func (o *EmailIntakeOrchestrator) Reject(ctx context.Context, tenantID, conversationID, version int) (*ent.EmailConversation, error) {
	updated, err := o.client.EmailConversation.UpdateOneID(conversationID).Where(emailconversation.TenantIDEQ(tenantID), emailconversation.VersionEQ(version)).SetStatus("REJECTED").AddVersion(1).Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("reject email conversation: %w", err)
	}
	return updated, nil
}

func (o *EmailIntakeOrchestrator) Override(ctx context.Context, tenantID, conversationID, version, actorID int, reason string) (*ent.EmailConversation, error) {
	if len(strings.TrimSpace(reason)) < 5 || actorID <= 0 {
		return nil, errors.New("override reason is required")
	}
	conversation, err := o.client.EmailConversation.Query().Where(emailconversation.IDEQ(conversationID), emailconversation.TenantIDEQ(tenantID)).WithMessages(func(q *ent.InboundEmailMessageQuery) { q.Order(ent.Desc(inboundemailmessage.FieldReceivedAt)).Limit(1) }).Only(ctx)
	if err != nil {
		return nil, err
	}
	if conversation.Version != version {
		return nil, errors.New("conversation was modified by another operator")
	}
	if conversation.SupportContractID == nil {
		return nil, errors.New("support contract must be selected before override")
	}
	fields, err := fieldsFromMap(conversation.CanonicalData)
	if err != nil {
		return nil, err
	}
	runtimeConfig := o.runtimeConfig(ctx, tenantID)
	if o.incidentService == nil || runtimeConfig.AutomationReporterUserID <= 0 {
		return nil, errors.New("incident automation reporter is not configured")
	}
	response, err := o.incidentService.CreateFromEmail(ctx, tenantID, service.EmailIncidentCommand{ConversationID: conversation.ID, SupportContractID: *conversation.SupportContractID, ReporterUserID: runtimeConfig.AutomationReporterUserID, AssignmentGroupID: runtimeConfig.DefaultAssignmentGroupID, Title: defaultString(fields.Title, "邮件报障"), Description: fields.Description, Impact: defaultString(fields.Impact, "medium"), Urgency: defaultString(fields.Urgency, "medium"), Category: "network", OverrideContract: true, OverrideReason: reason, Metadata: map[string]interface{}{"aiConfidence": fields.Confidence, "contractOverrideBy": actorID, "contractOverrideReason": reason}})
	if err != nil {
		return nil, err
	}
	updated, err := o.client.EmailConversation.UpdateOneID(conversation.ID).Where(emailconversation.TenantIDEQ(tenantID), emailconversation.VersionEQ(version)).SetStatus("INCIDENT_CREATED").AddVersion(1).Save(ctx)
	if err != nil {
		return nil, err
	}
	if len(conversation.Edges.Messages) > 0 {
		if err := o.enqueueIncidentCreatedReply(ctx, tenantID, updated, conversation.Edges.Messages[0], response.IncidentNumber); err != nil {
			return nil, err
		}
	}
	return updated, nil
}

func (o *EmailIntakeOrchestrator) updateConversation(ctx context.Context, conversation *ent.EmailConversation, status string, fields IntakeFields, resolution Resolution) (*ent.EmailConversation, error) {
	data, _ := fieldsToMap(fields)
	update := o.client.EmailConversation.UpdateOneID(conversation.ID).Where(emailconversation.TenantIDEQ(conversation.TenantID), emailconversation.VersionEQ(conversation.Version)).SetStatus(status).SetCanonicalData(data).SetMissingFields(resolution.MissingFields).SetConfidence(fields.Confidence).SetLastMessageAt(time.Now()).AddVersion(1)
	if resolution.CustomerID != 0 {
		update.SetCustomerID(resolution.CustomerID)
	}
	if resolution.BranchID != 0 {
		update.SetBranchID(resolution.BranchID)
	}
	if resolution.SupportContractID != 0 {
		update.SetSupportContractID(resolution.SupportContractID)
	}
	if resolution.SourceOrganizationID != 0 {
		update.SetSourceOrganizationID(resolution.SourceOrganizationID)
	}
	updated, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update email conversation: %w", err)
	}
	return updated, nil
}

func (o *EmailIntakeOrchestrator) findOrCreateConversation(ctx context.Context, tenantID int, email ReceivedEmail) (*ent.EmailConversation, error) {
	for _, reference := range append([]string{email.InReplyTo}, email.References...) {
		if reference == "" {
			continue
		}
		message, err := o.client.InboundEmailMessage.Query().Where(inboundemailmessage.TenantIDEQ(tenantID), inboundemailmessage.ExternalMessageIDEQ(reference)).WithConversation().First(ctx)
		if err == nil && message.Edges.Conversation != nil && sameMailbox(message.FromAddress, email.FromAddress) {
			return message.Edges.Conversation, nil
		}
	}
	if token := conversationTokenFromSubject(email.Subject); token != "" {
		existing, err := o.client.EmailConversation.Query().Where(emailconversation.TenantIDEQ(tenantID), emailconversation.ConversationTokenEQ(token)).Only(ctx)
		if err == nil && o.conversationHasSender(ctx, tenantID, existing.ID, email.FromAddress) {
			return existing, nil
		}
	}
	token, err := newConversationToken()
	if err != nil {
		return nil, err
	}
	return o.client.EmailConversation.Create().SetTenantID(tenantID).SetConversationToken(token).SetExternalThreadID(email.ExternalMessageID).SetLastMessageAt(email.ReceivedAt).Save(ctx)
}

// findOrCreateConversationTx creates or finds a conversation entirely within the provided transaction.
// It must be called from within an existing transaction so that the conversation and the
// InboundEmailMessage row land in the same DB transaction, satisfying the two-phase idempotent write.
func (o *EmailIntakeOrchestrator) findOrCreateConversationTx(ctx context.Context, tx *ent.Tx, tenantID int, email ReceivedEmail) (*ent.EmailConversation, error) {
	for _, reference := range append([]string{email.InReplyTo}, email.References...) {
		if reference == "" {
			continue
		}
		message, err := tx.InboundEmailMessage.Query().Where(inboundemailmessage.TenantIDEQ(tenantID), inboundemailmessage.ExternalMessageIDEQ(reference)).WithConversation().First(ctx)
		if err == nil && message.Edges.Conversation != nil && sameMailbox(message.FromAddress, email.FromAddress) {
			return message.Edges.Conversation, nil
		}
	}
	if token := conversationTokenFromSubject(email.Subject); token != "" {
		existing, err := tx.EmailConversation.Query().Where(emailconversation.TenantIDEQ(tenantID), emailconversation.ConversationTokenEQ(token)).Only(ctx)
		if err == nil && o.conversationHasSenderTx(ctx, tx, tenantID, existing.ID, email.FromAddress) {
			return existing, nil
		}
	}
	token, err := newConversationToken()
	if err != nil {
		return nil, err
	}
	return tx.EmailConversation.Create().SetTenantID(tenantID).SetConversationToken(token).SetExternalThreadID(email.ExternalMessageID).SetLastMessageAt(email.ReceivedAt).Save(ctx)
}

func (o *EmailIntakeOrchestrator) enqueueMissingInformationReply(ctx context.Context, tenantID int, conversation *ent.EmailConversation, message *ent.InboundEmailMessage, fields []string) error {
	if len(fields) == 0 {
		fields = []string{"customerName", "reportedContractNumber"}
	}
	labels := map[string]string{"customerName": "客户名称", "branchName": "分部名称", "reportedContractNumber": "合同号"}
	lines := make([]string, 0, len(fields))
	for i, field := range fields {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, labels[field]))
	}
	body := fmt.Sprintf("您好，已收到本次报障。\n\n尚缺少以下信息：\n%s\n\n请直接回复本邮件补充上述信息。\n\n会话编号：[%s]", strings.Join(lines, "\n"), conversation.ConversationToken)
	return o.enqueueReply(ctx, tenantID, conversation, message, "missing_information", body)
}

func (o *EmailIntakeOrchestrator) enqueueIncidentCreatedReply(ctx context.Context, tenantID int, conversation *ent.EmailConversation, message *ent.InboundEmailMessage, incidentNumber string) error {
	body := fmt.Sprintf("您好，您的报障已受理。\n\n工单编号：%s\n当前状态：已创建\n\n会话编号：[%s]", incidentNumber, conversation.ConversationToken)
	return o.enqueueReply(ctx, tenantID, conversation, message, "incident_created", body)
}

func (o *EmailIntakeOrchestrator) enqueueReply(ctx context.Context, tenantID int, conversation *ent.EmailConversation, message *ent.InboundEmailMessage, replyType, body string) error {
	to := message.FromAddress
	if _, err := mail.ParseAddress(to); err != nil {
		return errors.New("invalid reply address")
	}
	tx, err := o.client.Tx(ctx)
	if err != nil {
		return err
	}
	rollback := func(cause error) error { _ = tx.Rollback(); return cause }
	outbound, err := tx.EmailOutboundMessage.Create().SetTenantID(tenantID).SetConversationID(conversation.ID).SetMailboxInstanceKey(message.MailboxInstanceKey).SetReplyType(replyType).SetRevision(conversation.Version).SetToAddress(to).SetSubject("Re: " + message.Subject + " [" + conversation.ConversationToken + "]").SetBodyText(body).SetInReplyTo(message.ExternalMessageID).SetReferences(append(message.References, message.ExternalMessageID)).Save(ctx)
	if ent.IsConstraintError(err) {
		_ = tx.Rollback()
		return nil
	}
	if err != nil {
		return rollback(err)
	}
	_, err = commandbus.EnqueueTx(ctx, tx, commandbus.EnqueueRequest{TenantID: tenantID, CommandType: commandbus.CommandSendIntakeEmail, AggregateType: "email_outbound_message", AggregateID: outbound.ID, IdempotencyKey: fmt.Sprintf("email-outbound:%d", outbound.ID), Payload: map[string]interface{}{"outboundMessageId": outbound.ID}})
	if err != nil {
		return rollback(err)
	}
	if err = tx.Commit(); err != nil {
		return rollback(err)
	}
	return nil
}

func sameMailbox(a, b string) bool {
	pa, ea := mail.ParseAddress(a)
	pb, eb := mail.ParseAddress(b)
	return ea == nil && eb == nil && strings.EqualFold(pa.Address, pb.Address)
}

func (o *EmailIntakeOrchestrator) conversationHasSender(ctx context.Context, tenantID, conversationID int, sender string) bool {
	messages, err := o.client.InboundEmailMessage.Query().Where(inboundemailmessage.TenantIDEQ(tenantID), inboundemailmessage.ConversationIDEQ(conversationID)).All(ctx)
	if err != nil {
		return false
	}
	for _, message := range messages {
		if sameMailbox(message.FromAddress, sender) {
			return true
		}
	}
	return false
}

func (o *EmailIntakeOrchestrator) conversationHasSenderTx(ctx context.Context, tx *ent.Tx, tenantID, conversationID int, sender string) bool {
	messages, err := tx.InboundEmailMessage.Query().Where(inboundemailmessage.TenantIDEQ(tenantID), inboundemailmessage.ConversationIDEQ(conversationID)).All(ctx)
	if err != nil {
		return false
	}
	for _, message := range messages {
		if sameMailbox(message.FromAddress, sender) {
			return true
		}
	}
	return false
}

func (o *EmailIntakeOrchestrator) sourceOrganizationForSender(ctx context.Context, tenantID int, sender string) (string, int) {
	address, err := mail.ParseAddress(sender)
	if err != nil {
		return "", 0
	}
	parts := strings.Split(strings.ToLower(address.Address), "@")
	if len(parts) != 2 {
		return "", 0
	}
	items, err := o.client.SourceOrganization.Query().Where(sourceorganization.TenantIDEQ(tenantID), sourceorganization.StatusEQ("active")).All(ctx)
	if err != nil {
		return "", 0
	}
	var match *ent.SourceOrganization
	for _, item := range items {
		matched := false
		for _, candidate := range item.EmailAddresses {
			if strings.EqualFold(strings.TrimSpace(candidate), address.Address) {
				matched = true
			}
		}
		for _, domain := range item.EmailDomains {
			if strings.EqualFold(strings.TrimSpace(domain), parts[1]) {
				matched = true
			}
		}
		if matched {
			if match != nil {
				return "", 0
			}
			match = item
		}
	}
	if match == nil {
		return "", 0
	}
	return match.Name, match.ID
}

var tokenPattern = regexp.MustCompile(`\[([A-Z0-9]{12})\]`)

func conversationTokenFromSubject(subject string) string {
	match := tokenPattern.FindStringSubmatch(subject)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

func newConversationToken() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return strings.ToUpper(hex.EncodeToString(b)), nil
}

func fieldsToMap(fields IntakeFields) (map[string]interface{}, error) {
	data := map[string]interface{}{}
	raw, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(raw, &data)
	return data, err
}

func fieldsFromMap(data map[string]interface{}) (IntakeFields, error) {
	var fields IntakeFields
	raw, err := json.Marshal(data)
	if err != nil {
		return fields, err
	}
	err = json.Unmarshal(raw, &fields)
	return fields, err
}

func optionalInt(value int) *int {
	if value == 0 {
		return nil
	}
	return &value
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func limitRunes(value string, max int) string {
	r := []rune(value)
	if len(r) > max {
		return string(r[:max])
	}
	return value
}

func stringExtra(values map[string]interface{}, key string) string {
	value, _ := values[key].(string)
	return value
}

func numericUint64(value interface{}) (uint64, bool) {
	switch number := value.(type) {
	case uint32:
		return uint64(number), true
	case uint64:
		return number, true
	case int:
		return uint64(number), number >= 0
	case float64:
		return uint64(number), number >= 0
	default:
		return 0, false
	}
}

func mergeIntakeFields(current, incoming IntakeFields, sources map[string]interface{}) IntakeFields {
	canReplace := func(field string) bool { value, _ := sources[field].(string); return value != "human" }
	if incoming.SourceOrganizationName != "" && canReplace("sourceOrganizationName") {
		current.SourceOrganizationName = incoming.SourceOrganizationName
	}
	if incoming.CustomerName != "" && canReplace("customerName") {
		current.CustomerName = incoming.CustomerName
	}
	if incoming.BranchName != "" && canReplace("branchName") {
		current.BranchName = incoming.BranchName
	}
	if incoming.ReportedContractNumber != "" && canReplace("reportedContractNumber") {
		current.ReportedContractNumber = incoming.ReportedContractNumber
	}
	if incoming.Title != "" && canReplace("title") {
		current.Title = incoming.Title
	}
	if incoming.Description != "" && canReplace("description") {
		current.Description = incoming.Description
	}
	if incoming.OccurredAt != nil && canReplace("occurredAt") {
		current.OccurredAt = incoming.OccurredAt
	}
	if incoming.Impact != "" && canReplace("impact") {
		current.Impact = incoming.Impact
	}
	if incoming.Urgency != "" && canReplace("urgency") {
		current.Urgency = incoming.Urgency
	}
	if incoming.Confidence > 0 {
		current.Confidence = incoming.Confidence
	}
	return current
}

func copyMap(input map[string]interface{}) map[string]interface{} {
	output := make(map[string]interface{}, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func (o *EmailIntakeOrchestrator) runtimeConfig(ctx context.Context, tenantID int) OrchestratorConfig {
	config := o.config
	items, err := o.client.SystemConfig.Query().Where(systemconfig.TenantIDEQ(tenantID), systemconfig.KeyIn("email_intake.mode", "email_intake.automation_reporter_user_id", "email_intake.default_group_id"), systemconfig.DeletedAtIsNil()).All(ctx)
	if err != nil {
		return config
	}
	for _, item := range items {
		switch item.Key {
		case "email_intake.mode":
			mode := IntakeMode(item.Value)
			if mode == ModeObserveOnly || mode == ModeManualConfirm || mode == ModeAutoCreate {
				config.Mode = mode
			}
		case "email_intake.automation_reporter_user_id":
			if value, parseErr := strconv.Atoi(item.Value); parseErr == nil && value > 0 {
				config.AutomationReporterUserID = value
			}
		case "email_intake.default_group_id":
			if value, parseErr := strconv.Atoi(item.Value); parseErr == nil && value > 0 {
				config.DefaultAssignmentGroupID = &value
			}
		}
	}
	return config
}
