package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type LocalStorage struct {
	basePath string
}

func NewLocalStorage(basePath string) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, err
	}
	return &LocalStorage{basePath: basePath}, nil
}

func (s *LocalStorage) fullPath(key string) string {
	return filepath.Join(s.basePath, key)
}

func (s *LocalStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64) error {
	path := s.fullPath(key)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	return err
}

func (s *LocalStorage) Download(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	path := s.fullPath(key)

	info, err := os.Stat(path)
	if err != nil {
		return nil, 0, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}

	return file, info.Size(), nil
}

func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	return os.Remove(s.fullPath(key))
}

func (s *LocalStorage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := os.Stat(s.fullPath(key))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}