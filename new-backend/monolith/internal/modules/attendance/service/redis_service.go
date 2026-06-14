package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisService struct {
	client *redis.Client
}

func NewRedisService(client *redis.Client) *RedisService {
	return &RedisService{client: client}
}

// Session cache operations
func (s *RedisService) SetSessionCache(ctx context.Context, sessionID string, data map[string]any, ttl time.Duration) error {
	key := fmt.Sprintf("attendance:session:%s", sessionID)
	pipe := s.client.Pipeline()
	pipe.HSet(ctx, key, data)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	if err != nil {
		s.client.Expire(ctx, key, 6*time.Hour)
		return err
	}
	return nil
}

func (s *RedisService) GetSessionCache(ctx context.Context, sessionID string) (map[string]string, error) {
	key := fmt.Sprintf("attendance:session:%s", sessionID)
	return s.client.HGetAll(ctx, key).Result()
}

// Enrolled students set
func (s *RedisService) AddEnrolledStudents(ctx context.Context, sessionID string, studentIDs []uuid.UUID) error {
	key := fmt.Sprintf("attendance:session:%s:enrolled", sessionID)
	members := make([]any, len(studentIDs))
	for i, id := range studentIDs {
		members[i] = id.String()
	}
	return s.client.SAdd(ctx, key, members...).Err()
}

func (s *RedisService) IsStudentEnrolled(ctx context.Context, sessionID, studentID string) (bool, error) {
	key := fmt.Sprintf("attendance:session:%s:enrolled", sessionID)
	return s.client.SIsMember(ctx, key, studentID).Result()
}

// AddToBuffer claims a scan slot atomically and writes it to the buffer.
// Returns (true, nil) when the scan is newly recorded, (false, nil) when the
// student already scanned this session. SADD is used as the dedup primitive so
// the check-and-set is atomic (no TOCTOU race between concurrent scans).
func (s *RedisService) AddToBuffer(ctx context.Context, sessionID, studentID, data string, scannedTTL time.Duration) (bool, error) {
	bufferKey := fmt.Sprintf("attendance:buffer:%s", sessionID)
	scannedKey := fmt.Sprintf("attendance:scanned:%s", sessionID)

	added, err := s.client.SAdd(ctx, scannedKey, studentID).Result()
	if err != nil {
		return false, err
	}
	if added == 0 {
		return false, nil
	}

	pipe := s.client.Pipeline()
	pipe.HSet(ctx, bufferKey, studentID, data)
	pipe.Expire(ctx, scannedKey, scannedTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		// Roll back the claim so the student can retry.
		s.client.SRem(ctx, scannedKey, studentID)
		return false, err
	}
	return true, nil
}

func (s *RedisService) GetBuffer(ctx context.Context, sessionID string) (map[string]string, error) {
	key := fmt.Sprintf("attendance:buffer:%s", sessionID)
	return s.client.HGetAll(ctx, key).Result()
}

// Clear all session keys
func (s *RedisService) ClearSessionKeys(ctx context.Context, sessionID string) error {
	keys := []string{
		fmt.Sprintf("attendance:session:%s", sessionID),
		fmt.Sprintf("attendance:session:%s:enrolled", sessionID),
		fmt.Sprintf("attendance:buffer:%s", sessionID),
		fmt.Sprintf("attendance:scanned:%s", sessionID),
	}
	return s.client.Del(ctx, keys...).Err()
}

// Student summary cache
func (s *RedisService) InvalidateStudentSummary(ctx context.Context, studentID uuid.UUID, semester string) error {
	key := fmt.Sprintf("attendance:student:%s:summary:%s", studentID.String(), semester)
	return s.client.Del(ctx, key).Err()
}

// GetAllBufferKeys returns every session buffer key using SCAN.
// SCAN is cursor-based and non-blocking, unlike KEYS which is O(N) and
// stalls the Redis event loop on large keyspaces.
func (s *RedisService) GetAllBufferKeys(ctx context.Context) ([]string, error) {
	const pattern = "attendance:buffer:*"
	const scanCount = 100

	var keys []string
	iter := s.client.Scan(ctx, 0, pattern, scanCount).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

// HDelBufferFields removes the given student fields from a session buffer hash.
// Unlike ClearBuffer (DEL), this only removes the fields we successfully flushed,
// so records added concurrently and records that failed to persist are preserved
// for the next tick.
func (s *RedisService) HDelBufferFields(ctx context.Context, sessionID string, studentIDs []string) error {
	if len(studentIDs) == 0 {
		return nil
	}
	key := fmt.Sprintf("attendance:buffer:%s", sessionID)
	fields := make([]string, len(studentIDs))
	copy(fields, studentIDs)
	return s.client.HDel(ctx, key, fields...).Err()
}
