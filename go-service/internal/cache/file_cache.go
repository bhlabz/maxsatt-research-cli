package cache

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
)

type CacheEntry[T any] struct {
	Data      T         `json:"data"`
	CreatedAt time.Time `json:"created_at"`
	Checksum  string    `json:"checksum"`
}

type CacheService[T any] interface {
	Get(key string) (T, bool)
	Set(key string, data T) error
	GenerateKey(params ...interface{}) string
}

type FileCache[T any] struct {
	cacheDir string
}

func NewFileCache[T any](subDir string) *FileCache[T] {
	cacheDir := filepath.Join(properties.RootPath()+"/data", subDir)
	return &FileCache[T]{
		cacheDir: cacheDir,
	}
}

func (fc *FileCache[T]) GenerateKey(params ...interface{}) string {
	var keyData string
	for _, param := range params {
		keyData += fmt.Sprintf("%v_", param)
	}
	h := sha1.New()
	h.Write([]byte(keyData))
	return hex.EncodeToString(h.Sum(nil))
}

func (fc *FileCache[T]) Get(key string) (T, bool) {
	var zero T
	cacheFile := filepath.Join(fc.cacheDir, key+".json")
	
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return zero, false
	}
	
	var entry CacheEntry[T]
	if err := json.Unmarshal(data, &entry); err != nil {
		return zero, false
	}
	
	expectedChecksum := fc.calculateChecksum(entry.Data)
	if entry.Checksum != expectedChecksum {
		return zero, false
	}
	
	return entry.Data, true
}

func (fc *FileCache[T]) Set(key string, data T) error {
	if err := os.MkdirAll(fc.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %v", err)
	}
	
	entry := CacheEntry[T]{
		Data:      data,
		CreatedAt: time.Now(),
		Checksum:  fc.calculateChecksum(data),
	}
	
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %v", err)
	}
	
	cacheFile := filepath.Join(fc.cacheDir, key+".json")
	tmpFile := cacheFile + ".tmp"
	
	if err := os.WriteFile(tmpFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write temp cache file: %v", err)
	}
	
	if err := os.Rename(tmpFile, cacheFile); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to rename temp cache file: %v", err)
	}
	
	return nil
}

func (fc *FileCache[T]) calculateChecksum(data T) string {
	jsonData, _ := json.Marshal(data)
	hash := md5.Sum(jsonData)
	return hex.EncodeToString(hash[:])
}