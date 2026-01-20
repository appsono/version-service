package storage

import (
	"context"
	"io"
)

type Storage interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64) error
	Download(ctx context.Context, key string) (io.ReadCloser, int64, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

type FallbackStorage struct {
	primary   Storage
	fallback  Storage
}

func NewFallbackStorage(primary, fallback Storage) *FallbackStorage {
	return &FallbackStorage{
		primary:  primary,
		fallback: fallback,
	}
}

func (s *FallbackStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64) error {
	if s.primary != nil {
		if err := s.primary.Upload(ctx, key, reader, size); err == nil {
			return nil
		}
	}
	if s.fallback != nil {
		return s.fallback.Upload(ctx, key, reader, size)
	}
	return nil
}

func (s *FallbackStorage) Download(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	if s.primary != nil {
		rc, size, err := s.primary.Download(ctx, key)
		if err == nil {
			return rc, size, nil
		}
	}
	if s.fallback != nil {
		return s.fallback.Download(ctx, key)
	}
	return nil, 0, io.EOF
}

func (s *FallbackStorage) Delete(ctx context.Context, key string) error {
	var lastErr error
	if s.primary != nil {
		if err := s.primary.Delete(ctx, key); err != nil {
			lastErr = err
		}
	}
	if s.fallback != nil {
		if err := s.fallback.Delete(ctx, key); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (s *FallbackStorage) Exists(ctx context.Context, key string) (bool, error) {
	if s.primary != nil {
		exists, err := s.primary.Exists(ctx, key)
		if err == nil && exists {
			return true, nil
		}
	}
	if s.fallback != nil {
		return s.fallback.Exists(ctx, key)
	}
	return false, nil
}