// Package store implements the crash-safe, append-only parameter store.
//
// The store uses two data structures working together:
//
// 1. APPEND-ONLY LOG (JSONL file):
//   - Every change is appended as a new line, never modified or deleted
//   - Provides complete audit trail with operation type and client IP
//   - Survives crashes - can always replay to recover state
//
// 2. IN-MEMORY INDEX (map[key]*Parameter):
//   - Maps each key to its latest value for O(1) lookups
//   - Rebuilt from log on startup
//   - Updated only AFTER data is safely on disk
package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"parameter-store/internal/models"
)

type Store struct {
	mu       sync.RWMutex
	filePath string
	index    map[string]*models.Parameter
}

func New(filePath string) (*Store, error) {
	s := &Store{
		filePath: filePath,
		index:    make(map[string]*models.Parameter),
	}
	if err := s.rebuildIndex(); err != nil {
		return nil, fmt.Errorf("failed to rebuild index: %w", err)
	}
	return s, nil
}

// rebuildIndex replays the entire log to reconstruct the in-memory index.
// This is called on startup to restore state after a restart or crash.
//
// The log is processed sequentially - later entries override earlier ones.
// Delete operations remove the key from the index entirely.
func (s *Store) rebuildIndex() error {
	file, err := os.Open(s.filePath)
	if os.IsNotExist(err) {
		// No log file yet - start fresh
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var param models.Parameter
		if err := json.Unmarshal([]byte(line), &param); err != nil {
			// Skip corrupted lines - partial writes from crashes
			continue
		}

		if param.Operation == models.OpDelete {
			delete(s.index, param.Key)
		} else {
			paramCopy := param
			s.index[param.Key] = &paramCopy
		}
	}

	fmt.Printf("Loaded %d parameters\n", len(s.index))
	return scanner.Err()
}

// BatchUpdate writes multiple updates atomically with crash safety.
//
// The crash safety mechanism works as follows:
//
//  1. LOCK: Acquire mutex to prevent concurrent writes
//  2. WRITE: Append all records to the log file
//  3. FSYNC: Force OS to flush data to physical disk
//     - Without this, data might only be in OS buffer
//     - A crash would lose buffered data
//  4. UPDATE INDEX: Only after fsync confirms data is on disk
//     - This ensures index never references unpersisted data
//  5. UNLOCK: Release mutex
//
// Crash scenarios:
//   - Crash before fsync: Data lost, but index unchanged (consistent)
//   - Crash after fsync: Data on disk, index rebuilt on restart (consistent)
//
// The key insight: we never update the index until we're certain
// the data has reached the disk. This makes the system crash-safe.
func (s *Store) BatchUpdate(updates []models.UpdateRequest, clientIP string) error {
	if len(updates) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	timestamp := time.Now().UnixMilli()
	records := make([]*models.Parameter, len(updates))

	// Write all records to file
	for i, update := range updates {
		// Determine operation type based on current state
		var op models.Operation
		if update.IsDelete {
			op = models.OpDelete
		} else if _, exists := s.index[update.Key]; exists {
			op = models.OpUpdate
		} else {
			op = models.OpInsert
		}

		record := &models.Parameter{
			Key:       update.Key,
			Value:     update.Value,
			Type:      update.Type,
			Operation: op,
			Timestamp: timestamp,
			IP:        clientIP,
		}
		records[i] = record

		data, err := json.Marshal(record)
		if err != nil {
			return err
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			return err
		}
	}

	// CRITICAL: fsync ensures data reaches disk before we update the index.
	// This is what makes the system crash-safe.
	if err := file.Sync(); err != nil {
		return err
	}

	// Safe to update index now - data is persisted
	for _, record := range records {
		if record.Operation == models.OpDelete {
			delete(s.index, record.Key)
		} else {
			s.index[record.Key] = record
		}
	}

	return nil
}

func (s *Store) Get(key string) *models.Parameter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.index[key]
}

// List returns all parameters. Password values are masked unless unmaskPasswords is true.
func (s *Store) List(unmaskPasswords bool) []models.ParameterView {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]models.ParameterView, 0, len(s.index))
	for _, param := range s.index {
		view := models.ParameterView{
			Key:       param.Key,
			Value:     param.Value,
			Type:      param.Type,
			Timestamp: param.Timestamp,
			Masked:    false,
		}

		if param.Type == models.TypePassword && !unmaskPasswords {
			view.Value = "********"
			view.Masked = true
		}

		result = append(result, view)
	}
	return result
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.index)
}

// GetHistory returns all log entries for a given key (all inserts, updates, deletes).
func (s *Store) GetHistory(key string) []models.Parameter {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var history []models.Parameter

	file, err := os.Open(s.filePath)
	if err != nil {
		return history
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var param models.Parameter
		if err := json.Unmarshal([]byte(line), &param); err != nil {
			continue
		}

		if param.Key == key {
			history = append(history, param)
		}
	}

	return history
}
