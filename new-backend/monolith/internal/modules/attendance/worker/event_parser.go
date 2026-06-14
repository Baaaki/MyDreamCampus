package worker

import (
	"encoding/json"
	"fmt"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/dto"
)

// unwrapEventData unmarshals an outbox-shaped message into the caller's
// typed payload. The on-the-wire format wraps the typed body inside a
// BaseEvent envelope:
//
//	{ "event_id": ..., "event_type": ..., "timestamp": ..., "data": { ... } }
//
// Each handler used to repeat the same Unmarshal → Marshal → Unmarshal
// dance inline. Centralising it here:
//   - prevents drift between handlers (e.g. one forgetting to validate the
//     base envelope before reading data),
//   - lets event_parser_test.go exercise the parsing edges (bad envelope,
//     bad inner data, contract drift) without spinning up a consumer.
func unwrapEventData[T any](body []byte) (T, error) {
	var zero T

	var base dto.BaseEvent
	if err := json.Unmarshal(body, &base); err != nil {
		return zero, fmt.Errorf("invalid base envelope: %w", err)
	}

	dataBytes, err := json.Marshal(base.Data)
	if err != nil {
		return zero, fmt.Errorf("re-marshal of data field failed: %w", err)
	}

	var data T
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return zero, fmt.Errorf("typed data did not match contract: %w", err)
	}
	return data, nil
}
