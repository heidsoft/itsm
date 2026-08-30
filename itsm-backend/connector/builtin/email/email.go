package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"net/netip"
	"net/smtp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	imapclient "github.com/emersion/go-imap/client"
	"itsm-backend/connector"
)

var errAutomatedMessage = errors.New("automated email is not eligible for intake")

type Connector struct {
	cfg          connector.Config
	username     string
	password     string
	imapAddress  string
	smtpAddress  string
	imapTlsMode  string
	smtpTlsMode  string
	mailbox      string
	fromName     string
	pollInterval time.Duration
	handler      connector.PollingInboundHandler
	mu           sync.RWMutex
	startOnce    sync.Once
	cancel       context.CancelFunc
}

func init()           { connector.MustRegister(func() connector.Connector { return New() }) }
func New() *Connector { return &Connector{} }

func (c *Connector) Manifest() connector.Manifest {
	return connector.Manifest{Name: "email", Version: "1.1.1", Title: "IMAP/SMTP 邮箱", Provider: "standard", Type: connector.TypeEmail, Description: "通过安全 IMAP 接收报障邮件并通过 SMTP 回复，支持主流公网邮箱服务", Capabilities: []connector.Capability{connector.CapSendMessage, connector.CapReceiveMessage, connector.CapReplyMessage, connector.CapHealthCheck}, Tags: []string{"email", "imap", "smtp", "qq", "163", "gmail", "outlook", "exchange"}, IsOfficial: true, RequiredPermissions: []string{"connector:write", "email_intake:review"}, ConfigSchema: `{"type":"object","required":["username","password","imapHost","smtpHost"],"properties":{"username":{"type":"string","title":"邮箱账号"},"password":{"type":"string","format":"password","title":"授权码/密码"},"imapHost":{"type":"string","title":"IMAP 服务器"},"imapPort":{"type":"integer","default":993},"imapTlsMode":{"type":"string","enum":["ssl","starttls"],"default":"ssl"},"smtpHost":{"type":"string","title":"SMTP 服务器"},"smtpPort":{"type":"integer","default":465},"smtpTlsMode":{"type":"string","enum":["ssl","starttls"],"default":"ssl"},"mailbox":{"type":"string","default":"INBOX"},"pollIntervalSeconds":{"type":"integer","minimum":15,"default":30},"fromName":{"type":"string","title":"发件人名称"}}}`}
}

func (c *Connector) Init(_ context.Context, cfg connector.Config) error {
	username := strings.TrimSpace(cfg.Credentials["username"])
	password := cfg.Credentials["password"]
	if username == "" || password == "" {
		return errors.New("email username and app password are required")
	}
	imapHost := settingString(cfg.Settings, "imapHost", "imap.qq.com")
	smtpHost := settingString(cfg.Settings, "smtpHost", "smtp.qq.com")
	imapPort := settingInt(cfg.Settings, "imapPort", 993)
	smtpPort := settingInt(cfg.Settings, "smtpPort", 465)
	if imapPort < 1 || imapPort > 65535 || smtpPort < 1 || smtpPort > 65535 {
		return errors.New("invalid email port")
	}
	c.imapTlsMode = normalizeTlsMode(settingString(cfg.Settings, "imapTlsMode", "ssl"))
	c.smtpTlsMode = normalizeTlsMode(settingString(cfg.Settings, "smtpTlsMode", "ssl"))
	if c.imapTlsMode == "plain" || c.smtpTlsMode == "plain" {
		return errors.New("plaintext email transport is not supported")
	}
	if err := validateConfiguredHost(imapHost); err != nil {
		return fmt.Errorf("invalid IMAP host: %w", err)
	}
	if err := validateConfiguredHost(smtpHost); err != nil {
		return fmt.Errorf("invalid SMTP host: %w", err)
	}
	pollSeconds := settingInt(cfg.Settings, "pollIntervalSeconds", 30)
	if pollSeconds < 15 {
		pollSeconds = 15
	}
	c.cfg = cfg
	c.username = username
	c.password = password
	c.imapAddress = net.JoinHostPort(imapHost, strconv.Itoa(imapPort))
	c.smtpAddress = net.JoinHostPort(smtpHost, strconv.Itoa(smtpPort))
	c.mailbox = settingString(cfg.Settings, "mailbox", "INBOX")
	c.fromName = settingString(cfg.Settings, "fromName", "")
	c.pollInterval = time.Duration(pollSeconds) * time.Second
	return nil
}

func (c *Connector) SetInboundHandler(handler connector.PollingInboundHandler) {
	c.mu.Lock()
	c.handler = handler
	c.mu.Unlock()
}

func (c *Connector) Start(parent context.Context) error {
	var startErr error
	c.startOnce.Do(func() {
		if c.username == "" {
			startErr = errors.New("email connector is not initialized")
			return
		}
		ctx, cancel := context.WithCancel(parent)
		c.cancel = cancel
		go c.loop(ctx)
	})
	return startErr
}

func (c *Connector) loop(ctx context.Context) {
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()
	c.poll(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.poll(ctx)
		}
	}
}

func (c *Connector) dialIMAP() (*imapclient.Client, error) {
	host := hostOnly(c.imapAddress)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resolvedAddress, err := resolvePublicEndpoint(ctx, c.imapAddress)
	if err != nil {
		return nil, err
	}
	switch c.imapTlsMode {
	case "ssl":
		return imapclient.DialTLS(resolvedAddress, &tls.Config{MinVersion: tls.VersionTLS12, ServerName: host})
	case "starttls":
		cli, err := imapclient.Dial(resolvedAddress)
		if err != nil {
			return nil, err
		}
		if err := cli.StartTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: host}); err != nil {
			cli.Logout()
			return nil, err
		}
		return cli, nil
	default:
		return nil, errors.New("unsupported IMAP TLS mode")
	}
}

func (c *Connector) poll(ctx context.Context) {
	c.mu.RLock()
	handler := c.handler
	c.mu.RUnlock()
	if handler == nil {
		return
	}
	client, err := c.dialIMAP()
	if err != nil {
		return
	}
	defer client.Logout()
	if err = client.Login(c.username, c.password); err != nil {
		return
	}
	mailbox, err := client.Select(c.mailbox, false)
	if err != nil {
		return
	}
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	uids, err := client.UidSearch(criteria)
	if err != nil || len(uids) == 0 {
		return
	}
	set := new(imap.SeqSet)
	for _, uid := range uids {
		set.AddNum(uid)
	}
	section := &imap.BodySectionName{}
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- client.UidFetch(set, []imap.FetchItem{imap.FetchUid, imap.FetchEnvelope, section.FetchItem()}, messages)
	}()
	for message := range messages {
		if message == nil {
			continue
		}
		body := message.GetBody(section)
		if body == nil {
			continue
		}
		raw, readErr := io.ReadAll(io.LimitReader(body, 25*1024*1024+1))
		if readErr != nil || len(raw) > 25*1024*1024 {
			continue
		}
		inbound, parseErr := parseInbound(raw)
		if parseErr != nil {
			if errors.Is(parseErr, errAutomatedMessage) {
				one := new(imap.SeqSet)
				one.AddNum(message.Uid)
				_ = client.UidStore(one, imap.AddFlags, []interface{}{imap.SeenFlag}, nil)
			}
			continue
		}
		inbound.ConnectorName = "email"
		inbound.ConnectorType = connector.TypeEmail
		inbound.Channel = c.username
		inbound.MessageID = inboundMessageID(inbound)
		inbound.ReceivedAt = time.Now()
		if message.Envelope != nil && !message.Envelope.Date.IsZero() {
			inbound.ReceivedAt = message.Envelope.Date
		}
		if inbound.Extras == nil {
			inbound.Extras = map[string]interface{}{}
		}
		inbound.Extras["uid"] = message.Uid
		inbound.Extras["uidValidity"] = mailbox.UidValidity
		if err := handler(ctx, c.cfg.TenantID, cfgInstanceKey(c.cfg), inbound); err == nil {
			one := new(imap.SeqSet)
			one.AddNum(message.Uid)
			_ = client.UidStore(one, imap.AddFlags, []interface{}{imap.SeenFlag}, nil)
		}
	}
	<-done
}

func (c *Connector) Send(ctx context.Context, msg *connector.Message) error {
	if msg == nil {
		return errors.New("email message is required")
	}
	to, err := mail.ParseAddress(msg.Channel)
	if err != nil {
		return errors.New("invalid recipient address")
	}
	if strings.ContainsAny(msg.Title+msg.ID, "\r\n") {
		return errors.New("invalid email title")
	}
	fromAddr, err := mail.ParseAddress(c.username)
	if err != nil {
		return errors.New("invalid sender address")
	}
	if c.fromName != "" {
		fromAddr.Name = c.fromName
	}
	subject := mime.QEncoding.Encode("UTF-8", defaultString(msg.Title, "ITSM 通知"))
	messageID := strings.Trim(strings.TrimSpace(msg.ID), "<>")
	if messageID == "" {
		messageID = fmt.Sprintf("email-%d", time.Now().UnixNano())
	}
	message := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMessage-ID: <%s@itsm.local>\r\nAuto-Submitted: auto-generated\r\nX-ITSM-Auto-Reply: 1\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", fromAddr.String(), to.String(), subject, messageID, normalizeBody(msg.Content)))
	return c.sendSMTP(ctx, to.Address, message)
}

func (c *Connector) sendSMTP(ctx context.Context, to string, message []byte) error {
	host := hostOnly(c.smtpAddress)
	resolvedAddress, err := resolvePublicEndpoint(ctx, c.smtpAddress)
	if err != nil {
		return fmt.Errorf("resolve SMTP endpoint: %w", err)
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var conn net.Conn
	switch c.smtpTlsMode {
	case "ssl":
		conn, err = tls.DialWithDialer(dialer, "tcp", resolvedAddress, &tls.Config{MinVersion: tls.VersionTLS12, ServerName: host})
	case "starttls":
		conn, err = dialer.DialContext(ctx, "tcp", resolvedAddress)
	default:
		return errors.New("unsupported SMTP TLS mode")
	}
	if err != nil {
		return fmt.Errorf("connect SMTP: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(deadline(ctx, 15*time.Second))
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()
	if c.smtpTlsMode == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("SMTP server does not support STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: host}); err != nil {
			return err
		}
	}
	if ok, _ := client.Extension("AUTH"); !ok {
		return errors.New("SMTP server does not support authentication")
	}
	if err = client.Auth(smtp.PlainAuth("", c.username, c.password, host)); err != nil {
		return err
	}
	if err = client.Mail(c.username); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = writer.Write(message); err != nil {
		return err
	}
	return writer.Close()
}

func (c *Connector) HealthCheck(ctx context.Context) connector.HealthStatus {
	start := time.Now()
	status := connector.HealthStatus{CheckedAt: start}
	client, err := c.dialIMAP()
	if err == nil {
		err = client.Login(c.username, c.password)
		_ = client.Logout()
	}
	status.LatencyMs = time.Since(start).Milliseconds()
	status.OK = err == nil
	if err != nil {
		status.Message = err.Error()
	} else {
		status.Message = "IMAP authentication succeeded"
	}
	return status
}

func (c *Connector) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}

func parseInbound(raw []byte) (*connector.InboundMessage, error) {
	message, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	from := message.Header.Get("From")
	if _, err = mail.ParseAddress(from); err != nil {
		return nil, errors.New("invalid From header")
	}
	if isAutomatedMessage(message.Header) {
		return nil, errAutomatedMessage
	}
	subject, _ := new(mime.WordDecoder).DecodeHeader(message.Header.Get("Subject"))
	plain, htmlBody, err := readMailBody(message.Header, message.Body)
	if err != nil {
		return nil, err
	}
	toAddresses := headerAddresses(message.Header, "To")
	replyTo := ""
	if addresses := headerAddresses(message.Header, "Reply-To"); len(addresses) > 0 {
		replyTo = addresses[0]
	}
	references := strings.Fields(message.Header.Get("References"))
	extras := map[string]interface{}{"subject": subject, "toAddresses": toAddresses, "replyToAddress": replyTo, "inReplyTo": strings.TrimSpace(message.Header.Get("In-Reply-To")), "references": references, "htmlBody": htmlBody}
	extrasRaw, _ := json.Marshal(extras)
	return &connector.InboundMessage{UserID: from, Content: plain, Type: "email", Raw: raw, Extras: map[string]interface{}{"headers": json.RawMessage(extrasRaw), "subject": subject, "toAddresses": toAddresses, "replyToAddress": replyTo, "inReplyTo": strings.TrimSpace(message.Header.Get("In-Reply-To")), "references": references, "htmlBody": htmlBody, "externalMessageId": strings.TrimSpace(message.Header.Get("Message-ID"))}}, nil
}

func isAutomatedMessage(header mail.Header) bool {
	autoSubmitted := strings.ToLower(strings.TrimSpace(header.Get("Auto-Submitted")))
	precedence := strings.ToLower(strings.TrimSpace(header.Get("Precedence")))
	contentType := strings.ToLower(header.Get("Content-Type"))
	from := strings.ToLower(header.Get("From"))
	return (autoSubmitted != "" && autoSubmitted != "no") || precedence == "bulk" || precedence == "list" || header.Get("X-Autoreply") != "" || strings.Contains(contentType, "multipart/report") || strings.Contains(from, "mailer-daemon") || strings.Contains(from, "postmaster")
}

func readMailBody(header mail.Header, body io.Reader) (string, string, error) {
	mediaType, params, _ := mime.ParseMediaType(header.Get("Content-Type"))
	if strings.HasPrefix(mediaType, "multipart/") {
		reader := multipart.NewReader(body, params["boundary"])
		var plain, htmlBody string
		for {
			part, err := reader.NextPart()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return "", "", err
			}
			partType, _, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
			content, readErr := io.ReadAll(io.LimitReader(part, 5*1024*1024))
			if readErr != nil {
				continue
			}
			if partType == "text/plain" && plain == "" {
				plain = string(content)
			}
			if partType == "text/html" && htmlBody == "" {
				htmlBody = string(content)
			}
		}
		if plain == "" {
			plain = stripHTML(htmlBody)
		}
		return plain, htmlBody, nil
	}
	content, err := io.ReadAll(io.LimitReader(body, 10*1024*1024))
	if err != nil {
		return "", "", err
	}
	if mediaType == "text/html" {
		return stripHTML(string(content)), string(content), nil
	}
	return string(content), "", nil
}

func headerAddresses(header mail.Header, key string) []string {
	values, err := header.AddressList(key)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.Address)
	}
	return out
}

func inboundMessageID(message *connector.InboundMessage) string {
	if value, ok := message.Extras["externalMessageId"].(string); ok {
		return value
	}
	return ""
}

func cfgInstanceKey(cfg connector.Config) string {
	return fmt.Sprintf("%d/%s/%s", cfg.TenantID, cfg.Name, cfg.Provider)
}

func normalizeTlsMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "starttls":
		return "starttls"
	case "plain", "none":
		return "plain"
	default:
		return "ssl"
	}
}

func settingString(settings map[string]interface{}, key, fallback string) string {
	if value, ok := settings[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func settingInt(settings map[string]interface{}, key string, fallback int) int {
	switch value := settings[key].(type) {
	case int:
		return value
	case float64:
		return int(value)
	case string:
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func hostOnly(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return address
	}
	return host
}

func validateConfiguredHost(host string) error {
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	if host == "" || strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".localhost") {
		return errors.New("local mail hosts are not allowed")
	}
	if ip, err := netip.ParseAddr(host); err == nil && !isPublicMailIP(ip) {
		return errors.New("private or local mail addresses are not allowed")
	}
	return nil
}

func resolvePublicEndpoint(ctx context.Context, address string) (string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", errors.New("invalid mail endpoint")
	}
	if err := validateConfiguredHost(host); err != nil {
		return "", err
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil || len(ips) == 0 {
		return "", errors.New("mail host cannot be resolved")
	}
	for _, ip := range ips {
		if !isPublicMailIP(ip) {
			return "", errors.New("mail host resolves to a private or local address")
		}
	}
	return net.JoinHostPort(ips[0].String(), port), nil
}

func isPublicMailIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	for _, blocked := range []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"),
		netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"),
		netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("240.0.0.0/4"),
		netip.MustParsePrefix("2001:db8::/32"),
	} {
		if blocked.Contains(ip) {
			return false
		}
	}
	return true
}

func deadline(ctx context.Context, fallback time.Duration) time.Time {
	if value, ok := ctx.Deadline(); ok {
		return value
	}
	return time.Now().Add(fallback)
}

func normalizeBody(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\n", "\r\n")
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func stripHTML(value string) string {
	var out strings.Builder
	inTag := false
	for _, r := range value {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			out.WriteRune(' ')
			continue
		}
		if !inTag {
			out.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(out.String()), " ")
}
