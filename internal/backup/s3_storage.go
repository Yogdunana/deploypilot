package backup

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// s3Storage implements StorageProvider using S3-compatible API.
// Supports AWS S3, Aliyun OSS, Tencent COS, MinIO, and any S3-compatible storage.
type s3Storage struct {
	endpoint  string
	region    string
	bucket   string
	prefix    string
	accessKey string
	secretKey string
	client    *http.Client
}

// NewS3Storage creates a new S3-compatible storage provider.
func NewS3Storage(cfg StorageConfig) (StorageProvider, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("s3 storage: endpoint is required")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("s3 storage: bucket is required")
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("s3 storage: access_key and secret_key are required")
	}

	endpoint := cfg.Endpoint
	if cfg.UseSSL && !strings.HasPrefix(endpoint, "https://") {
		if strings.HasPrefix(endpoint, "http://") {
			endpoint = "https://" + endpoint[7:]
		} else {
			endpoint = "https://" + endpoint
		}
	} else if !cfg.UseSSL && !strings.HasPrefix(endpoint, "http") {
		endpoint = "http://" + endpoint
	}

	prefix := cfg.Prefix
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}

	return &s3Storage{
		endpoint:  endpoint,
		region:    region,
		bucket:   cfg.Bucket,
		prefix:    prefix,
		accessKey: cfg.AccessKey,
		secretKey: cfg.SecretKey,
		client: &http.Client{
			Timeout: 10 * time.Minute, // large file uploads
		},
	}, nil
}

func (s *s3Storage) Type() StorageType {
	switch {
	case strings.Contains(s.endpoint, "aliyuncs.com"):
		return StorageOSS
	case strings.Contains(s.endpoint, "myqcloud.com"):
		return StorageCOS
	case strings.Contains(s.endpoint, "minio"):
		return StorageMinIO
	default:
		return StorageS3
	}
}

// Upload uploads an object to S3 using PutObject.
func (s *s3Storage) Upload(ctx context.Context, key string, reader io.Reader, size int64) error {
	fullKey := s.prefix + key

	// Read all content into memory for signing (simple approach for backup files)
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return fmt.Errorf("s3 upload: failed to read content: %w", err)
	}
	content := buf.Bytes()

	date := time.Now().UTC().Format("20060102")
	datetime := time.Now().UTC().Format("20060102T150405Z")
	host := s.bucket + "." + s.endpointURL().Host

	// Build canonical request
	canonicalURI := "/" + s.bucket + "/" + fullKey
	contentSHA256 := sha256Hex(content)

	canonicalHeaders := fmt.Sprintf("content-type:application/octet-stream\nhost:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		host, contentSHA256, datetime)
	signedHeaders := "content-type;host;x-amz-content-sha256;x-amz-date"

	canonicalRequest := fmt.Sprintf("PUT\n%s\n\n%s\n%s\n%s",
		canonicalURI, "", canonicalHeaders, signedHeaders, contentSHA256)

	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", date, s.region)
	stringToSign := aws4StringToSign(datetime, credentialScope, sha256Hex([]byte(canonicalRequest)))

	signature := s.sign(datetime, stringToSign)

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		s.accessKey, credentialScope, signedHeaders, signature)

	reqURL := fmt.Sprintf("%s://%s/%s", s.endpointURL().Scheme, host, fullKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reqURL, bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("s3 upload: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Host", host)
	req.Header.Set("x-amz-content-sha256", contentSHA256)
	req.Header.Set("x-amz-date", datetime)
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(content)))

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("s3 upload: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("s3 upload failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	slog.Info("uploaded backup to cloud storage", "key", fullKey, "size", len(content))
	return nil
}

// Download downloads an object from S3.
func (s *s3Storage) Download(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	fullKey := s.prefix + key

	date := time.Now().UTC().Format("20060102")
	datetime := time.Now().UTC().Format("20060102T150405Z")
	host := s.bucket + "." + s.endpointURL().Host

	canonicalURI := "/" + s.bucket + "/" + fullKey
	contentSHA256 := sha256Hex([]byte{})

	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		host, contentSHA256, datetime)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	canonicalRequest := fmt.Sprintf("GET\n%s\n\n%s\n%s\n%s",
		canonicalURI, "", canonicalHeaders, signedHeaders, contentSHA256)

	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", date, s.region)
	stringToSign := aws4StringToSign(datetime, credentialScope, sha256Hex([]byte(canonicalRequest)))
	signature := s.sign(datetime, stringToSign)

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		s.accessKey, credentialScope, signedHeaders, signature)

	reqURL := fmt.Sprintf("%s://%s/%s", s.endpointURL().Scheme, host, fullKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("s3 download: failed to create request: %w", err)
	}

	req.Header.Set("Host", host)
	req.Header.Set("x-amz-content-sha256", contentSHA256)
	req.Header.Set("x-amz-date", datetime)
	req.Header.Set("Authorization", authHeader)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("s3 download: request failed: %w", err)
	}

	if resp.StatusCode >= 300 {
		_ = resp.Body.Close()
		return nil, 0, fmt.Errorf("s3 download failed (HTTP %d)", resp.StatusCode)
	}

	return resp.Body, resp.ContentLength, nil
}

// Delete removes an object from S3.
func (s *s3Storage) Delete(ctx context.Context, key string) error {
	fullKey := s.prefix + key

	date := time.Now().UTC().Format("20060102")
	datetime := time.Now().UTC().Format("20060102T150405Z")
	host := s.bucket + "." + s.endpointURL().Host

	canonicalURI := "/" + s.bucket + "/" + fullKey
	contentSHA256 := sha256Hex([]byte{})

	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		host, contentSHA256, datetime)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	canonicalRequest := fmt.Sprintf("DELETE\n%s\n\n%s\n%s\n%s",
		canonicalURI, "", canonicalHeaders, signedHeaders, contentSHA256)

	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", date, s.region)
	stringToSign := aws4StringToSign(datetime, credentialScope, sha256Hex([]byte(canonicalRequest)))
	signature := s.sign(datetime, stringToSign)

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		s.accessKey, credentialScope, signedHeaders, signature)

	reqURL := fmt.Sprintf("%s://%s/%s", s.endpointURL().Scheme, host, fullKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return fmt.Errorf("s3 delete: failed to create request: %w", err)
	}

	req.Header.Set("Host", host)
	req.Header.Set("x-amz-content-sha256", contentSHA256)
	req.Header.Set("x-amz-date", datetime)
	req.Header.Set("Authorization", authHeader)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("s3 delete: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 && resp.StatusCode != 404 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("s3 delete failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	slog.Info("deleted backup from cloud storage", "key", fullKey)
	return nil
}

// List lists objects with the given prefix.
func (s *s3Storage) List(ctx context.Context, prefix string) ([]StorageObject, error) {
	listPrefix := s.prefix + prefix
	if listPrefix != "" && !strings.HasSuffix(listPrefix, "/") {
		listPrefix += "/"
	}

	date := time.Now().UTC().Format("20060102")
	datetime := time.Now().UTC().Format("20060102T150405Z")
	host := s.bucket + "." + s.endpointURL().Host

	query := url.Values{}
	query.Set("list-type", "2")
	query.Set("prefix", listPrefix)
	query.Set("max-keys", "1000")

	canonicalURI := "/" + s.bucket + "/"
	escapedQuery := query.Encode()
	contentSHA256 := sha256Hex([]byte{})

	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		host, contentSHA256, datetime)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	canonicalRequest := fmt.Sprintf("GET\n%s\n%s\n%s\n%s%s",
		canonicalURI, escapedQuery, canonicalHeaders, signedHeaders, contentSHA256)

	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", date, s.region)
	stringToSign := aws4StringToSign(datetime, credentialScope, sha256Hex([]byte(canonicalRequest)))
	signature := s.sign(datetime, stringToSign)

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		s.accessKey, credentialScope, signedHeaders, signature)

	reqURL := fmt.Sprintf("%s://%s/?%s", s.endpointURL().Scheme, host, escapedQuery)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("s3 list: failed to create request: %w", err)
	}

	req.Header.Set("Host", host)
	req.Header.Set("x-amz-content-sha256", contentSHA256)
	req.Header.Set("x-amz-date", datetime)
	req.Header.Set("Authorization", authHeader)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3 list: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("s3 list failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var listResult s3ListResultV2
	if err := xml.NewDecoder(resp.Body).Decode(&listResult); err != nil {
		return nil, fmt.Errorf("s3 list: failed to decode response: %w", err)
	}

	var objects []StorageObject
	for _, item := range listResult.Contents {
		// Strip the configured prefix from the key
		key := strings.TrimPrefix(item.Key, s.prefix)
		objects = append(objects, StorageObject{
			Key:          key,
			Size:         item.Size,
			LastModified: item.LastModified,
			ETag:         strings.Trim(item.ETag, "\""),
		})
	}

	// Sort by last modified (newest first)
	sort.Slice(objects, func(i, j int) bool {
		return objects[i].LastModified.After(objects[j].LastModified)
	})

	return objects, nil
}

// Exists checks if an object exists in S3 using HeadObject.
func (s *s3Storage) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := s.prefix + key

	date := time.Now().UTC().Format("20060102")
	datetime := time.Now().UTC().Format("20060102T150405Z")
	host := s.bucket + "." + s.endpointURL().Host

	canonicalURI := "/" + s.bucket + "/" + fullKey
	contentSHA256 := sha256Hex([]byte{})

	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		host, contentSHA256, datetime)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	canonicalRequest := fmt.Sprintf("HEAD\n%s\n\n%s\n%s%s",
		canonicalURI, "", canonicalHeaders, signedHeaders, contentSHA256)

	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", date, s.region)
	stringToSign := aws4StringToSign(datetime, credentialScope, sha256Hex([]byte(canonicalRequest)))
	signature := s.sign(datetime, stringToSign)

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		s.accessKey, credentialScope, signedHeaders, signature)

	reqURL := fmt.Sprintf("%s://%s/%s", s.endpointURL().Scheme, host, fullKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, reqURL, nil)
	if err != nil {
		return false, fmt.Errorf("s3 exists: failed to create request: %w", err)
	}

	req.Header.Set("Host", host)
	req.Header.Set("x-amz-content-sha256", contentSHA256)
	req.Header.Set("x-amz-date", datetime)
	req.Header.Set("Authorization", authHeader)

	resp, err := s.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("s3 exists: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == 200, nil
}

// endpointURL returns the parsed endpoint URL.
func (s *s3Storage) endpointURL() *url.URL {
	u, _ := url.Parse(s.endpoint)
	if u == nil {
		u = &url.URL{Host: s.endpoint}
	}
	return u
}

// sign generates an AWS4-HMAC-SHA256 signature.
func (s *s3Storage) sign(datetime, stringToSign string) string {
	date := datetime[:8]
	hmacKey := hmacSHA256([]byte("AWS4"+s.secretKey), []byte(date))
	hmacKey = hmacSHA256(hmacKey, []byte(s.region))
	hmacKey = hmacSHA256(hmacKey, []byte("s3"))
	hmacKey = hmacSHA256(hmacKey, []byte("aws4_request"))
	return hex.EncodeToString(hmacSHA256(hmacKey, []byte(stringToSign)))
}

// hmacSHA256 computes HMAC-SHA256.
func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// sha256Hex returns the hex-encoded SHA-256 hash.
func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// aws4StringToSign builds the AWS4 string to sign.
func aws4StringToSign(datetime, credentialScope, hashedCanonicalRequest string) string {
	return fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s", datetime, credentialScope, hashedCanonicalRequest)
}

// S3 ListObjectsV2 response types
type s3ListResultV2 struct {
	XMLName xml.Name `xml:"ListBucketResult"`
	Contents []s3Object
}

type s3Object struct {
	Key          string    `xml:"Key"`
	Size         int64     `xml:"Size"`
	LastModified time.Time `xml:"LastModified"`
	ETag         string    `xml:"ETag"`
}
