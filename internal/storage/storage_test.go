package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github-stars-notify/internal/github"
)

func TestStorageBasic(t *testing.T) {
	// Create temporary directory for testing
	testDir := "./test_storage"
	defer os.RemoveAll(testDir)

	// Create file storage
	storage := NewFileStorage(testDir)
	ctx := context.Background()

	// Test initialization
	err := storage.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Error("Storage directory was not created")
	}

	// Test saving and loading data
	stargazers := []github.Stargazer{
		{
			Login:     "testuser1",
			ID:        123,
			NodeID:    "node123",
			AvatarURL: "https://github.com/testuser1.png",
			StarredAt: time.Now(),
		},
		{
			Login:     "testuser2",
			ID:        456,
			NodeID:    "node456",
			AvatarURL: "https://github.com/testuser2.png",
			StarredAt: time.Now(),
		},
	}

	// Save data
	err = storage.Save(ctx, "facebook", "react", stargazers)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load data
	repoData, err := storage.Load(ctx, "facebook", "react")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if repoData.Owner != "facebook" {
		t.Errorf("Expected owner 'facebook', got %s", repoData.Owner)
	}
	if repoData.Repo != "react" {
		t.Errorf("Expected repo 'react', got %s", repoData.Repo)
	}
	if len(repoData.Stargazers) != 2 {
		t.Errorf("Expected 2 stargazers, got %d", len(repoData.Stargazers))
	}

	// Test getting new stargazers
	newStargazers := []github.Stargazer{
		stargazers[0], // existing
		stargazers[1], // existing
		{
			Login:     "newuser",
			ID:        789,
			NodeID:    "node789",
			AvatarURL: "https://github.com/newuser.png",
			StarredAt: time.Now(),
		}, // new
	}

	detected, err := storage.GetNewStargazers(ctx, "facebook", "react", newStargazers)
	if err != nil {
		t.Fatalf("GetNewStargazers failed: %v", err)
	}

	if len(detected) != 1 {
		t.Errorf("Expected 1 new stargazer, got %d", len(detected))
	}
	if detected[0].Login != "newuser" {
		t.Errorf("Expected new stargazer 'newuser', got %s", detected[0].Login)
	}

	// Test loading non-existent repository
	emptyData, err := storage.Load(ctx, "nonexistent", "repo")
	if err != nil {
		t.Fatalf("Load of non-existent repo failed: %v", err)
	}
	if len(emptyData.Stargazers) != 0 {
		t.Errorf("Expected empty stargazers for non-existent repo, got %d", len(emptyData.Stargazers))
	}

	// Test close
	err = storage.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestStorageFromConfig(t *testing.T) {
	// Test creating storage from config
	cfg := StorageConfig{
		Type: "file",
		Path: "./test_config_storage",
	}
	defer os.RemoveAll(cfg.Path)

	storage, err := NewStorageFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewStorageFromConfig failed: %v", err)
	}

	if storage == nil {
		t.Error("Expected storage to be created")
	}

	// Test unsupported type
	badCfg := StorageConfig{
		Type: "unsupported",
	}
	_, err = NewStorageFromConfig(badCfg)
	if err == nil {
		t.Error("Expected error for unsupported storage type")
	}
}
