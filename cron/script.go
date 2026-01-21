package cron

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"jiayou_backend_spider/config"
	"jiayou_backend_spider/utils"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"github.com/rclone/rclone/fs/hash"
)

type DScript struct {
	Payload           any           `json:"payload"`
	CachePath         string        `json:"cache_path"`
	CacheExt          string        `json:"cache_ext" default:"js"`
	CacheLockBlocking time.Duration `json:"cache_lock_blocking" default:"1s"`
	UseCache          bool          `json:"use_cache" default:"true"`
	Hash              hash.Type     `json:"hash"`
}

func (script *DScript) sum(data []byte) ([]byte, error) {
	var hasher = hash.NewMultiHasher()
	_, err := io.Copy(hasher, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return hasher.Sum(script.Hash)
}
func (script *DScript) SetPayload(payload any) {
	script.Payload = payload
}
func (script *DScript) SetHash(hash hash.Type) {
	script.Hash = hash
}
func (script *DScript) CacheFullPath(name string) string {
	return filepath.Clean(script.CachePath + "/" + name + script.CacheExt)
}
func (script *DScript) CacheExists(name string) bool {
	if script.CachePath == "" {
		return false
	}
	return utils.Exists(script.CacheFullPath(name))
}
func (script *DScript) CacheData(name string) ([]byte, error) {
	if !script.CacheExists(name) {
		return nil, errors.New("cache not existed")
	}
	var fileLck = flock.New(script.CacheFullPath(name))
	var lck, err = fileLck.TryLockContext(context.Background(), script.CacheLockBlocking)
	if err != nil {
		return nil, fmt.Errorf("get file lock failed, %w", err)
	}
	if lck {
		defer fileLck.Close()
		fp, err := os.OpenFile(script.CacheFullPath(name), os.O_RDONLY, 0644)
		if err != nil {
			return nil, err
		}
		defer fp.Close()
		data, err := io.ReadAll(fp)
		if err != nil {
			return nil, err
		}
		err = fileLck.Unlock()
		if err != nil {
			return nil, fmt.Errorf("release file lck failed, %w", err)
		}
		return data, nil
	}
	return nil, errors.New("get file lock timeout")

}
func (script *DScript) CacheCompare(name string, data []byte) (bool, error) {
	cache, err := script.CacheData(name)
	if err != nil {
		return false, err
	}
	srcSum, err := script.sum(cache)
	if err != nil {
		return false, err
	}
	destSum, err := script.sum(data)
	if err != nil {
		return false, err
	}
	return bytes.Compare(srcSum, destSum) == 0, nil
}
func (script *DScript) CacheUpdate(name string, data []byte) error {
	if !script.CacheExists(name) {
		return errors.New("cache not existed")
	}
	var fileLck = flock.New(script.CacheFullPath(name))
	var lck, err = fileLck.TryLockContext(context.Background(), script.CacheLockBlocking)
	if err != nil {
		return fmt.Errorf("get file lock failed, %w", err)
	}
	if lck {
		defer fileLck.Close()
		fp, err := os.OpenFile(script.CacheFullPath(name), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer fp.Close()
		_, err = io.Copy(fp, bytes.NewReader(data))
		if err != nil {
			return err
		}
		err = fileLck.Unlock()
		if err != nil {
			return fmt.Errorf("release file lck failed, %w", err)
		}
		return nil
	}
	return errors.New("get file lock timeout")
}

type LocalScript struct {
	DScript
	Name   string `json:"name" validate:"required"`
	Script string `json:"script" validate:"required"`
}

func (local *LocalScript) SetName(name string) {
	local.Name = name
}
func (local *LocalScript) SetScript(script string) {
	local.Script = script
}

type StorageScript struct {
	DScript
	Name    string `json:"name" validate:"required"`
	Storage string `json:"storage" validate:"required"`
}

func (storage *StorageScript) SetName(name string) {
	storage.Name = name
}
func (storage *StorageScript) SetStorage(stor string) {
	storage.Storage = stor
}

func NewLocalScript(script string) *LocalScript {
	return config.TryValidate(&LocalScript{Script: script})
}
func NewStorageScript(storage string) *StorageScript {
	return config.TryValidate(&StorageScript{Storage: storage})
}
