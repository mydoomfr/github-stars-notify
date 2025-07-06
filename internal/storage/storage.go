package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github-stars-notify/internal/errors"
	"github-stars-notify/internal/github"
)

// Storage defines the interface for data persistence
type Storage interface {
	// Initialize prepares the storage for use
	Initialize(ctx context.Context) error

	// Load loads the stored data for a repository
	Load(ctx context.Context, owner, repo string) (*RepoData, error)

	// Save saves the data for a repository
	Save(ctx context.Context, owner, repo string, stargazers []github.Stargazer) error

	// GetNewStargazers compares current stargazers with previous data and returns new ones
	GetNewStargazers(ctx context.Context, owner, repo string, currentStargazers []github.Stargazer) ([]github.Stargazer, error)

	// GetLastCheckTime returns the last check time for a repository
	GetLastCheckTime(ctx context.Context, owner, repo string) (time.Time, error)

	// Close closes the storage and cleans up resources
	Close() error
}

// RepoData represents stored data for a repository
type RepoData struct {
	Owner        string             `json:"owner"`
	Repo         string             `json:"repo"`
	LastCheck    time.Time          `json:"last_check"`
	Stargazers   []github.Stargazer `json:"stargazers"`
	PreviousData *RepoData          `json:"previous_data,omitempty"`
}

// FileStorage implements Storage interface using file system
type FileStorage struct {
	dataDir string
	mutex   sync.RWMutex
}

// NewFileStorage creates a new file-based storage instance
func NewFileStorage(dataDir string) *FileStorage {
	if dataDir == "" {
		dataDir = "./data"
	}

	return &FileStorage{
		dataDir: dataDir,
	}
}

// NewStorage creates a new storage instance (legacy function for backward compatibility)
func NewStorage(dataDir string) Storage {
	return NewFileStorage(dataDir)
}

// Initialize creates the data directory if it doesn't exist
func (s *FileStorage) Initialize(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return errors.NewStorageError("initialize", s.dataDir,
			"failed to create data directory", err)
	}
	return nil
}

// Load loads the stored data for a repository
func (s *FileStorage) Load(ctx context.Context, owner, repo string) (*RepoData, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	filename := s.getFilename(owner, repo)

	// Check if context is cancelled
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// Return empty data if file doesn't exist
		return &RepoData{
			Owner:      owner,
			Repo:       repo,
			LastCheck:  time.Time{},
			Stargazers: []github.Stargazer{},
		}, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.NewStorageError("load", filename,
			"failed to read data file", err)
	}

	var repoData RepoData
	if err := json.Unmarshal(data, &repoData); err != nil {
		return nil, errors.NewStorageError("load", filename,
			"failed to unmarshal data", err)
	}

	return &repoData, nil
}

// Save saves the data for a repository
func (s *FileStorage) Save(ctx context.Context, owner, repo string, stargazers []github.Stargazer) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	filename := s.getFilename(owner, repo)

	// Check if context is cancelled
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Load existing data to preserve as previous
	existingData, err := s.loadUnsafe(owner, repo)
	if err != nil {
		return errors.NewStorageError("save", filename,
			"failed to load existing data", err)
	}

	// Create new data with current stargazers
	newData := &RepoData{
		Owner:      owner,
		Repo:       repo,
		LastCheck:  time.Now(),
		Stargazers: stargazers,
	}

	// Preserve previous data if it exists and has stargazers
	if len(existingData.Stargazers) > 0 {
		newData.PreviousData = &RepoData{
			Owner:      existingData.Owner,
			Repo:       existingData.Repo,
			LastCheck:  existingData.LastCheck,
			Stargazers: existingData.Stargazers,
		}
	}

	// Marshal and save
	data, err := json.MarshalIndent(newData, "", "  ")
	if err != nil {
		return errors.NewStorageError("save", filename,
			"failed to marshal data", err)
	}

	// Write to temporary file first, then rename (atomic write)
	tempFile := filename + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return errors.NewStorageError("save", tempFile,
			"failed to write temporary file", err)
	}

	if err := os.Rename(tempFile, filename); err != nil {
		// Clean up temporary file on error
		os.Remove(tempFile)
		return errors.NewStorageError("save", filename,
			"failed to rename temporary file", err)
	}

	return nil
}

// GetNewStargazers compares current stargazers with previous data and returns new ones
func (s *FileStorage) GetNewStargazers(ctx context.Context, owner, repo string, currentStargazers []github.Stargazer) ([]github.Stargazer, error) {
	repoData, err := s.Load(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to load repo data: %w", err)
	}

	// If no previous data, all stargazers are new
	if len(repoData.Stargazers) == 0 {
		return currentStargazers, nil
	}

	// Check if context is cancelled
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Create a map of existing stargazers for fast lookup
	existingStargazers := make(map[int64]bool)
	for _, sg := range repoData.Stargazers {
		existingStargazers[sg.ID] = true
	}

	// Find new stargazers
	var newStargazers []github.Stargazer
	for _, sg := range currentStargazers {
		if !existingStargazers[sg.ID] {
			newStargazers = append(newStargazers, sg)
		}
	}

	return newStargazers, nil
}

// GetLastCheckTime returns the last check time for a repository
func (s *FileStorage) GetLastCheckTime(ctx context.Context, owner, repo string) (time.Time, error) {
	repoData, err := s.Load(ctx, owner, repo)
	if err != nil {
		return time.Time{}, err
	}
	return repoData.LastCheck, nil
}

// Close closes the storage and cleans up resources
func (s *FileStorage) Close() error {
	// File storage doesn't need cleanup
	return nil
}

// getFilename generates the filename for a repository's data
func (s *FileStorage) getFilename(owner, repo string) string {
	return filepath.Join(s.dataDir, fmt.Sprintf("%s_%s.json", owner, repo))
}

// loadUnsafe loads data without acquiring a lock (for internal use)
func (s *FileStorage) loadUnsafe(owner, repo string) (*RepoData, error) {
	filename := s.getFilename(owner, repo)

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// Return empty data if file doesn't exist
		return &RepoData{
			Owner:      owner,
			Repo:       repo,
			LastCheck:  time.Time{},
			Stargazers: []github.Stargazer{},
		}, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.NewStorageError("load", filename,
			"failed to read data file", err)
	}

	var repoData RepoData
	if err := json.Unmarshal(data, &repoData); err != nil {
		return nil, errors.NewStorageError("load", filename,
			"failed to unmarshal data", err)
	}

	return &repoData, nil
}

// StorageConfig holds configuration for creating storage instances
type StorageConfig struct {
	Type string
	Path string
}

// NewStorageFromConfig creates a storage instance from configuration
func NewStorageFromConfig(cfg StorageConfig) (Storage, error) {
	switch cfg.Type {
	case "file", "":
		return NewFileStorage(cfg.Path), nil
	default:
		return nil, errors.NewStorageError("create", "",
			fmt.Sprintf("unsupported storage type: %s", cfg.Type), nil)
	}
}
