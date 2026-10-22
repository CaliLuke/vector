package assets

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithyauth "github.com/aws/smithy-go/auth"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// StorageConfig captures the S3 settings used by Django's S3Storage. Endpoint
// is the server-side endpoint; PublicEndpoint is used only to construct links
// returned to clients.
type StorageConfig struct {
	AccessKeyID      string
	SecretAccessKey  string
	Bucket           string
	Region           string
	Endpoint         string
	PublicEndpoint   string
	PathStyle        bool
	UseMinIO         bool
	MinIOEndpointSSL bool
	SignedURLTTL     time.Duration
}

// StorageConfigFromEnv maps Plane's existing Django storage settings. The
// PLANE_ASSETS_* variables are explicit overrides for isolated test services.
func StorageConfigFromEnv() (StorageConfig, error) {
	config := StorageConfig{
		AccessKeyID:      firstEnv("PLANE_ASSETS_S3_ACCESS_KEY_ID", "AWS_ACCESS_KEY_ID"),
		SecretAccessKey:  firstEnv("PLANE_ASSETS_S3_SECRET_ACCESS_KEY", "AWS_SECRET_ACCESS_KEY"),
		Bucket:           firstEnv("PLANE_ASSETS_S3_BUCKET", "AWS_S3_BUCKET_NAME"),
		Region:           firstEnv("PLANE_ASSETS_S3_REGION", "AWS_REGION"),
		Endpoint:         firstEnv("PLANE_ASSETS_S3_INTERNAL_ENDPOINT", "AWS_S3_ENDPOINT_URL", "MINIO_ENDPOINT_URL"),
		PublicEndpoint:   firstEnv("PLANE_ASSETS_S3_PUBLIC_ENDPOINT", "AWS_S3_PUBLIC_ENDPOINT_URL"),
		UseMinIO:         os.Getenv("USE_MINIO") == "1",
		MinIOEndpointSSL: os.Getenv("MINIO_ENDPOINT_SSL") == "1",
		SignedURLTTL:     time.Hour,
	}
	// Django passes os.environ.get("AWS_REGION") to boto3: an absent value
	// uses the provider chain, while an explicitly empty value stays empty.
	if _, explicitRegion := os.LookupEnv("AWS_REGION"); config.Region == "" && !explicitRegion {
		defaults, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithDefaultRegion("us-east-1"))
		if err != nil {
			return StorageConfig{}, fmt.Errorf("resolve assets S3 default region: %w", err)
		}
		config.Region = defaults.Region
	}
	if value := os.Getenv("PLANE_ASSETS_S3_PATH_STYLE"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return StorageConfig{}, fmt.Errorf("parse PLANE_ASSETS_S3_PATH_STYLE: %w", err)
		}
		config.PathStyle = parsed
	} else {
		config.PathStyle = config.UseMinIO || config.Endpoint != ""
	}
	if ttl := os.Getenv("SIGNED_URL_EXPIRATION"); ttl != "" {
		seconds, err := strconv.Atoi(ttl)
		if err != nil || seconds < 1 {
			return StorageConfig{}, fmt.Errorf("SIGNED_URL_EXPIRATION must be a positive integer number of seconds")
		}
		config.SignedURLTTL = time.Duration(seconds) * time.Second
	}
	if config.Bucket == "" || config.Region == "" && config.Endpoint == "" {
		return StorageConfig{}, fmt.Errorf("assets S3 requires a bucket and either a region or a custom endpoint")
	}
	return config, nil
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}

// Storage is the FileAsset object-store boundary.
type Storage struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	awsConfig     aws.Config
	bucket        string
	ttl           time.Duration
	config        StorageConfig
}

func NewStorage(config StorageConfig) (*Storage, error) {
	if config.Bucket == "" || config.Region == "" && config.Endpoint == "" {
		return nil, fmt.Errorf("assets S3 requires a bucket and either a region or a custom endpoint")
	}
	if config.SignedURLTTL <= 0 {
		config.SignedURLTTL = time.Hour
	}
	awsConfig, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(config.Region))
	if err != nil {
		return nil, fmt.Errorf("load AWS configuration: %w", err)
	}
	// Preserve an explicitly empty region accepted by boto3 for custom endpoints.
	awsConfig.Region = config.Region
	if (config.AccessKeyID == "") != (config.SecretAccessKey == "") {
		return nil, fmt.Errorf("assets S3 access key and secret key must be configured together")
	}
	if config.AccessKeyID != "" {
		awsConfig.Credentials = aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(config.AccessKeyID, config.SecretAccessKey, ""))
	}
	internalClient := newS3Client(awsConfig, config.Endpoint, config.PathStyle)
	presignEndpoint := config.PublicEndpoint
	if presignEndpoint == "" {
		presignEndpoint = config.Endpoint
	}
	return &Storage{
		client:        internalClient,
		presignClient: s3.NewPresignClient(newS3Client(awsConfig, presignEndpoint, config.PathStyle)),
		awsConfig:     awsConfig,
		bucket:        config.Bucket,
		ttl:           config.SignedURLTTL,
		config:        config,
	}, nil
}

func newS3Client(config aws.Config, endpoint string, pathStyle bool) *s3.Client {
	return s3.NewFromConfig(config, func(options *s3.Options) {
		options.UsePathStyle = pathStyle
		if endpoint != "" {
			options.BaseEndpoint = aws.String(endpoint)
			if config.Region == "" {
				options.EndpointResolverV2 = emptyRegionEndpointResolver{s3.NewDefaultEndpointResolverV2()}
			}
		}
	})
}

// Boto3 permits an empty signing region with a custom S3 endpoint (including
// Plane's default local configuration). The SDK endpoint rules require a
// region even for a fixed endpoint. Resolve the URL with a placeholder region,
// then retain the source region in the SDK's first-class signing properties.
// Actual AWS endpoints continue to use the normal resolver unchanged.
type emptyRegionEndpointResolver struct{ s3.EndpointResolverV2 }

func (resolver emptyRegionEndpointResolver) ResolveEndpoint(ctx context.Context, parameters s3.EndpointParameters) (smithyendpoints.Endpoint, error) {
	parameters.Region = aws.String("us-east-1")
	endpoint, err := resolver.EndpointResolverV2.ResolveEndpoint(ctx, parameters)
	if err != nil {
		return endpoint, err
	}
	options, _ := smithyauth.GetAuthOptions(&endpoint.Properties)
	for _, option := range options {
		if option.SchemeID == "sigv4" || option.SchemeID == smithyauth.SchemeIDSigV4 {
			smithyhttp.SetSigV4SigningRegion(&option.SignerProperties, "")
		}
	}
	return endpoint, nil
}

// PresignEndpointForRequest retains Django's request-host fallback for local
// MinIO deployments where the browser-facing host differs from the API host.
func (storage *Storage) PresignEndpointForRequest(requestScheme, requestHost string) string {
	if storage.config.PublicEndpoint != "" {
		return storage.config.PublicEndpoint
	}
	if storage.config.UseMinIO && requestHost != "" {
		protocol := requestScheme
		if storage.config.MinIOEndpointSSL {
			protocol = "https"
		}
		if protocol == "" {
			protocol = "http"
		}
		return protocol + "://" + requestHost
	}
	return storage.config.Endpoint
}

// PresignPost creates the S3 POST form Django returns for direct uploads.
func (storage *Storage) PresignPost(ctx context.Context, key, contentType string, maxBytes int64, requestScheme, requestHost string) (string, map[string]string, error) {
	client := storage.presignClient
	if endpoint := storage.PresignEndpointForRequest(requestScheme, requestHost); endpoint != storage.config.PublicEndpoint && endpoint != storage.config.Endpoint {
		client = s3.NewPresignClient(newS3Client(storage.awsConfig, endpoint, storage.config.PathStyle))
	}
	post, err := client.PresignPostObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(storage.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, func(options *s3.PresignPostOptions) {
		options.Expires = storage.ttl
		options.PostPresigner = botoCompatiblePostPresigner{}
		options.Conditions = []any{
			map[string]string{"bucket": storage.bucket},
			[]any{"content-length-range", int64(1), maxBytes},
			map[string]string{"Content-Type": contentType},
			map[string]string{"key": key},
		}
	})
	if err != nil {
		return "", nil, err
	}
	post.Values["Content-Type"] = contentType
	return post.URL, post.Values, nil
}

// botoCompatiblePostPresigner preserves the field casing and policy-condition
// order produced by Django's boto3 S3Storage. The AWS SDK's default POST
// presigner is valid, but emits X-Amz-* fields and leading SigV4 conditions
// that differ from the source contract.
type botoCompatiblePostPresigner struct{}

func (botoCompatiblePostPresigner) PresignPost(credentials aws.Credentials, bucket, key, region, service string, signingTime time.Time, conditions []any, expirationTime time.Time, _ ...func(*v4.SignerOptions)) (map[string]string, error) {
	credentialScope := signingTime.UTC().Format("20060102") + "/" + region + "/" + service + "/aws4_request"
	credential := credentials.AccessKeyID + "/" + credentialScope
	algorithm := "AWS4-HMAC-SHA256"
	date := signingTime.UTC().Format("20060102T150405Z")
	policyConditions := make([]any, 0, len(conditions)+6)
	policyConditions = append(policyConditions, conditions...)
	policyConditions = append(policyConditions,
		map[string]string{"bucket": bucket},
		map[string]string{"key": key},
		map[string]string{"x-amz-algorithm": algorithm},
		map[string]string{"x-amz-credential": credential},
		map[string]string{"x-amz-date": date},
	)
	fields := map[string]string{
		"key":              key,
		"x-amz-algorithm":  algorithm,
		"x-amz-credential": credential,
		"x-amz-date":       date,
	}
	if credentials.SessionToken != "" {
		fields["x-amz-security-token"] = credentials.SessionToken
		policyConditions = append(policyConditions, map[string]string{"x-amz-security-token": credentials.SessionToken})
	}
	policy, err := json.Marshal(map[string]any{
		"expiration": expirationTime.UTC().Format("2006-01-02T15:04:05Z"),
		"conditions": policyConditions,
	})
	if err != nil {
		return nil, fmt.Errorf("encode S3 POST policy: %w", err)
	}
	encodedPolicy := base64.StdEncoding.EncodeToString(policy)
	fields["policy"] = encodedPolicy
	fields["x-amz-signature"] = signPostPolicy(credentials.SecretAccessKey, signingTime, region, service, encodedPolicy)
	return fields, nil
}

func signPostPolicy(secret string, signingTime time.Time, region, service, encodedPolicy string) string {
	date := signingTime.UTC().Format("20060102")
	dateKey := hmacSHA256([]byte("AWS4"+secret), date)
	regionKey := hmacSHA256(dateKey, region)
	serviceKey := hmacSHA256(regionKey, service)
	signingKey := hmacSHA256(serviceKey, "aws4_request")
	return hex.EncodeToString(hmacSHA256(signingKey, encodedPolicy))
}

func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

// PresignGet returns a signed GET URL with Django-compatible response
// disposition. The caller is responsible for the HTTP 302 Location response.
func (storage *Storage) PresignGet(ctx context.Context, key, disposition string, filename *string, requestScheme, requestHost string) (string, error) {
	endpoint := storage.PresignEndpointForRequest(requestScheme, requestHost)
	client := storage.presignClient
	if endpoint != storage.config.PublicEndpoint && endpoint != storage.config.Endpoint {
		client = s3.NewPresignClient(newS3Client(storage.awsConfig, endpoint, storage.config.PathStyle))
	}
	if filename == nil {
		generated := randomHex32()
		filename = &generated
	}
	contentDisposition := contentDisposition(disposition, filename)
	request, err := client.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(storage.bucket),
		Key:                        aws.String(key),
		ResponseContentDisposition: aws.String(contentDisposition),
	}, s3.WithPresignExpires(storage.ttl))
	if err != nil {
		return "", err
	}
	return request.URL, nil
}

func (storage *Storage) Copy(ctx context.Context, sourceKey, destinationKey string) error {
	_, err := storage.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(storage.bucket),
		CopySource: aws.String(storage.bucket + "/" + encodeCopySourceKey(sourceKey)),
		Key:        aws.String(destinationKey),
	})
	return err
}

func (storage *Storage) Head(ctx context.Context, key string) (*s3.HeadObjectOutput, error) {
	return storage.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(storage.bucket), Key: aws.String(key)})
}

func (storage *Storage) Delete(ctx context.Context, key string) error {
	_, err := storage.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(storage.bucket), Key: aws.String(key)})
	return err
}

// Put stores a legacy multipart FileAsset before its database row is saved,
// matching Django's FileField storage write during model creation.
func (storage *Storage) Put(ctx context.Context, key, contentType string, contents []byte) error {
	_, err := storage.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(storage.bucket), Key: aws.String(key), Body: bytes.NewReader(contents), ContentType: aws.String(contentType),
	})
	return err
}

func (storage *Storage) Bucket() string { return storage.bucket }

func encodeCopySourceKey(key string) string {
	segments := strings.Split(key, "/")
	for index, segment := range segments {
		segments[index] = quoteS3Path(segment)
	}
	return strings.Join(segments, "/")
}

func contentDisposition(disposition string, filename *string) string {
	if filename == nil {
		generated := randomHex32()
		filename = &generated
	}
	if *filename == "" {
		return disposition
	}
	return disposition + "; filename*=UTF-8''" + quoteS3Path(*filename)
}

// quoteS3Path matches urllib.parse.quote's default safe set while retaining
// slashes used to delimit object-key components.
func quoteS3Path(value string) string {
	const hexDigits = "0123456789ABCDEF"
	var builder strings.Builder
	for index := 0; index < len(value); index++ {
		character := value[index]
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || strings.ContainsRune("_.~-", rune(character)) || character == '/' {
			builder.WriteByte(character)
			continue
		}
		builder.WriteByte('%')
		builder.WriteByte(hexDigits[character>>4])
		builder.WriteByte(hexDigits[character&0x0f])
	}
	return builder.String()
}

func randomHex32() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(value)
}
