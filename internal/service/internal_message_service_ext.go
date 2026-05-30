package service

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	v11 "admin/api/gen/internal_message/v1"
	"admin/internal/server"
	ssetransport "github.com/chnxq/xkitpkg/transport/sse"
)

const internalMessageEventName = "notification"

// TODO: add InternalMessageService-specific hooks, helpers, and hand-written business logic here.
// Add InternalMessageService-specific hooks and helpers here.
// This file is created once and is never overwritten by xkit.

func (s *InternalMessageService) listAllUserIDs(ctx context.Context) ([]uint32, error) {
	if s == nil || s.internalMessageRepo == nil {
		return nil, v11.ErrorBadRequest("target_all is not supported in current configuration")
	}
	return s.internalMessageRepo.ListAllUserIDs(ctx)
}

func (s *InternalMessageService) publishNotificationEvents(
	ctx context.Context,
	recipientIDs []uint32,
	messageID uint32,
	title string,
	content string,
) {
	server := s.resolveSSEServer()
	if server == nil || len(recipientIDs) == 0 {
		return
	}

	payload := map[string]any{
		"event":     internalMessageEventName,
		"messageId": messageID,
		"title":     title,
		"content":   content,
		"at":        time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	event := &ssetransport.Event{
		Event: []byte(internalMessageEventName),
		Data:  data,
	}

	for _, userID := range recipientIDs {
		if userID == 0 {
			continue
		}
		server.Publish(ctx, ssetransport.StreamID(strconv.FormatUint(uint64(userID), 10)), event)
	}
}

func (s *InternalMessageService) resolveSSEServer() *ssetransport.Server {
	if s == nil {
		return nil
	}
	return server.SharedSSEServer()
}

func normalizeRequiredText(value string) string {
	return strings.TrimSpace(value)
}

func dedupeUint32(values []uint32) []uint32 {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[uint32]struct{}, len(values))
	result := make([]uint32, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
