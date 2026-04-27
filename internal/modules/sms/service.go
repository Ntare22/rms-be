package sms

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/api/logger"
	"rms-be/internal/integrations/egosms"
	"rms-be/internal/middleware"
)

var reTemplateVar = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)

type Service struct {
	client         egosms.Client
	log            logger.Logger
	defaultSenderID string
}

func NewService(client egosms.Client, log logger.Logger, defaultSenderID string) *Service {
	return &Service{client: client, log: log, defaultSenderID: strings.TrimSpace(defaultSenderID)}
}

func (s *Service) Send(ctx context.Context, req *SendSMSRequest) (*SendSMSResponse, error) {
	start := time.Now().UTC()
	to, err := NormalizePhone(req.To)
	if err != nil {
		return nil, apierrors.New(400, apierrors.KindValidation, "invalid_phone", err.Error())
	}
	res, err := s.client.Send(ctx, egosms.SendRequest{
		To:        to,
		Message:   strings.TrimSpace(req.Message),
		SenderID:  chooseSender(req.SenderID, s.defaultSenderID),
		Reference: strings.TrimSpace(req.Reference),
	})
	latency := time.Since(start).Milliseconds()
	if err != nil {
		s.log.Error("sms send failed", "destinations", 1, "sender_id", mask(s.defaultSenderID), "outcome", "fail", "latency_ms", latency, "request_id", middleware.RequestIDFromRequest(ctx), "error", err.Error())
		return nil, mapProviderError(err)
	}
	out := SMSSendItemResult{
		To:                 to,
		Success:            res.Success,
		ProviderStatusCode: res.ProviderStatusCode,
		ProviderMessage:    res.ProviderMessage,
		MessageID:          res.MessageID,
		ReferenceID:        firstNonEmpty(res.ReferenceID, strings.TrimSpace(req.Reference)),
	}
	if !res.Success {
		s.log.Warn("sms send rejected", "destinations", 1, "sender_id", mask(s.defaultSenderID), "outcome", "fail", "latency_ms", latency, "request_id", middleware.RequestIDFromRequest(ctx), "provider_status_code", res.ProviderStatusCode)
		return &SendSMSResponse{Item: out}, apierrors.New(502, apierrors.KindInternal, "sms_provider_error", "SMS provider rejected the message")
	}
	s.log.Info("sms send success", "destinations", 1, "sender_id", mask(s.defaultSenderID), "outcome", "success", "latency_ms", latency, "request_id", middleware.RequestIDFromRequest(ctx))
	return &SendSMSResponse{Item: out}, nil
}

func (s *Service) BulkSend(ctx context.Context, req *BulkSendSMSRequest) (*BulkSendSMSResponse, error) {
	results := make([]SMSSendItemResult, 0, len(req.Recipients))
	sent := 0
	failed := 0
	for _, raw := range req.Recipients {
		itemReq := &SendSMSRequest{
			To:        raw,
			Message:   req.Message,
			SenderID:  req.SenderID,
			Reference: req.Reference,
		}
		out, err := s.Send(ctx, itemReq)
		if err != nil && out == nil {
			to := strings.TrimSpace(raw)
			if n, nerr := NormalizePhone(to); nerr == nil {
				to = n
			}
			results = append(results, SMSSendItemResult{
				To:              to,
				Success:         false,
				ProviderMessage: err.Error(),
			})
			failed++
			continue
		}
		results = append(results, out.Item)
		if out.Item.Success {
			sent++
		} else {
			failed++
		}
	}
	return &BulkSendSMSResponse{
		Total:   len(req.Recipients),
		Sent:    sent,
		Failed:  failed,
		Results: results,
	}, nil
}

func (s *Service) SendTemplate(ctx context.Context, req *SendTemplateSMSRequest) (*SendSMSResponse, error) {
	msg, err := RenderTemplate(req.Template, req.Variables)
	if err != nil {
		return nil, apierrors.New(400, apierrors.KindValidation, "invalid_template", err.Error())
	}
	return s.Send(ctx, &SendSMSRequest{
		To:       req.To,
		Message:  msg,
		SenderID: req.SenderID,
	})
}

func (s *Service) Health(ctx context.Context) (*SMSHealthResponse, error) {
	start := time.Now().UTC()
	out, err := s.client.Health(ctx)
	if err != nil {
		s.log.Error("sms health failed", "latency_ms", time.Since(start).Milliseconds(), "request_id", middleware.RequestIDFromRequest(ctx), "error", err.Error())
		return nil, mapProviderError(err)
	}
	return &SMSHealthResponse{
		Healthy:            out.Healthy,
		ProviderStatusCode: out.ProviderStatusCode,
		ProviderMessage:    out.ProviderMessage,
		CheckedAt:          time.Now().UTC(),
	}, nil
}

func RenderTemplate(template string, vars map[string]string) (string, error) {
	t := strings.TrimSpace(template)
	if t == "" {
		return "", fmt.Errorf("template is required")
	}
	out := reTemplateVar.ReplaceAllStringFunc(t, func(m string) string {
		matches := reTemplateVar.FindStringSubmatch(m)
		if len(matches) < 2 {
			return m
		}
		key := strings.TrimSpace(matches[1])
		if v, ok := vars[key]; ok {
			return v
		}
		return m
	})
	if strings.Contains(out, "{{") && strings.Contains(out, "}}") {
		return "", fmt.Errorf("template contains unresolved variables")
	}
	return out, nil
}

func NormalizePhone(in string) (string, error) {
	v := strings.TrimSpace(in)
	if v == "" {
		return "", fmt.Errorf("phone is required")
	}
	v = strings.ReplaceAll(v, " ", "")
	v = strings.ReplaceAll(v, "-", "")
	v = strings.ReplaceAll(v, "(", "")
	v = strings.ReplaceAll(v, ")", "")
	if strings.HasPrefix(v, "00") {
		v = "+" + strings.TrimPrefix(v, "00")
	}
	if !strings.HasPrefix(v, "+") {
		// Fallback heuristic: treat leading 0 numbers as Kenya local.
		if strings.HasPrefix(v, "0") {
			v = "+254" + strings.TrimPrefix(v, "0")
		} else {
			v = "+" + v
		}
	}
	for _, r := range strings.TrimPrefix(v, "+") {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("phone must contain digits only")
		}
	}
	digits := strings.TrimPrefix(v, "+")
	if len(digits) < 8 || len(digits) > 15 {
		return "", fmt.Errorf("phone must be in E.164 format")
	}
	return v, nil
}

func chooseSender(override, fallback string) string {
	if v := strings.TrimSpace(override); v != "" {
		return v
	}
	return strings.TrimSpace(fallback)
}

func mapProviderError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return apierrors.New(504, apierrors.KindInternal, "sms_timeout", "SMS provider timeout")
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return apierrors.New(504, apierrors.KindInternal, "sms_timeout", "SMS provider timeout")
	}
	return apierrors.New(502, apierrors.KindInternal, "sms_provider_error", "SMS provider request failed")
}

func firstNonEmpty(v ...string) string {
	for _, x := range v {
		if strings.TrimSpace(x) != "" {
			return strings.TrimSpace(x)
		}
	}
	return ""
}

func mask(v string) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return ""
	}
	if len(s) <= 2 {
		return "**"
	}
	return s[:1] + strings.Repeat("*", len(s)-2) + s[len(s)-1:]
}
