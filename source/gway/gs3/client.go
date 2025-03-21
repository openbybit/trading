package gs3

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/container"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const (
	defaultBucket = "defaultBucket"
	defaultPath   = "/tmp/s3"
)

// Client s3 client interface
type Client interface {
	Download(context.Context, string, time.Time) ([]byte, error)
}

// Session s3 session
type client struct {
	accessKeyID  string
	accessSecret string
	region       string
	path         string
	bucket       string // user specified
	cache        container.ConcurrentMap

	downloader *s3manager.Downloader
	rawSession *session.Session
}

type cacheEntry struct {
	checksum string
	lastTime time.Time
}

// NewClient new s3 session
func NewClient(keyID, secret, region string, opts ...Options) (Client, error) {
	if keyID == "" || secret == "" {
		return nil, errors.New("aws s3's id or secret is nil")
	}

	op := &option{}
	for _, opt := range opts {
		if opt != nil {
			opt(op)
		}
	}

	if op.path == "" {
		op.path = defaultPath
	}
	if op.bucket == "" {
		op.bucket = defaultBucket
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region:      aws.String(region),
			Credentials: credentials.NewStaticCredentials(keyID, secret, ""),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create aws session failed: %w", err)
	}

	return &client{
		path:         op.path,
		bucket:       op.bucket,
		region:       region,
		accessKeyID:  keyID,
		accessSecret: secret,
		rawSession:   sess,
		downloader:   s3manager.NewDownloader(sess),
		cache:        container.NewConcurrentMap(),
	}, nil
}

// Download file from s3
func (s *client) Download(ctx context.Context, key string, since time.Time) ([]byte, error) {
	if old, ok := s.cache.Get(key); ok && old != nil {
		entry := old.(cacheEntry)
		if !since.IsZero() && !since.After(entry.lastTime) {
			return nil, nil
		}
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	if !since.IsZero() {
		input.IfModifiedSince = &since
	}

	writer := aws.NewWriteAtBuffer([]byte{})
	_, err := s.downloader.DownloadWithContext(ctx, writer, input)
	if err != nil {
		return nil, err
	}

	s.cache.Set(key, cacheEntry{
		lastTime: time.Now(),
		checksum: toHex(writer.Bytes()),
	})
	return writer.Bytes(), nil
}

func toHex(bs []byte) string {
	h := md5.New()
	h.Write(bs)
	return hex.EncodeToString(h.Sum(nil))
}
