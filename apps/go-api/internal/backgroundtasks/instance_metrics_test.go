package backgroundtasks

import (
	"context"
	"crypto/tls"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	colmetricpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestInstanceMetricsBeatIntervalMatchesCeleryFallbacks(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want time.Duration
	}{
		{name: "default", raw: "", want: 360 * time.Minute},
		{name: "configured", raw: "5", want: 5 * time.Minute},
		{name: "invalid", raw: "bad", want: 360 * time.Minute},
		{name: "zero", raw: "0", want: 360 * time.Minute},
		{name: "too large", raw: "10000001", want: 360 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("METRICS_PUSH_INTERVAL_MINUTES", tt.raw)
			if got := instanceMetricsBeatIntervalFromEnv(); got != tt.want {
				t.Fatalf("interval = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestInstanceMetricsEndpointParsing(t *testing.T) {
	tests := []struct {
		name, raw, wantGRPC, wantHTTP string
	}{
		{name: "https host", raw: "https://telemetry.example.test", wantGRPC: "telemetry.example.test:443", wantHTTP: "https://telemetry.example.test/v1/metrics"},
		{name: "scheme-less host and port", raw: "telemetry.example.test:4318", wantGRPC: "telemetry.example.test:4318", wantHTTP: "https://telemetry.example.test:4318/v1/metrics"},
		{name: "http path", raw: "http://collector.example.test/otlp/", wantGRPC: "collector.example.test:4317", wantHTTP: "http://collector.example.test/otlp/v1/metrics"},
		{name: "IPv6 host", raw: "https://[2001:db8::1]", wantGRPC: "[2001:db8::1]:443", wantHTTP: "https://[2001:db8::1]/v1/metrics"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grpc, err := instanceMetricsGRPCEndpoint(tt.raw)
			if err != nil {
				t.Fatal(err)
			}
			if grpc != tt.wantGRPC {
				t.Fatalf("gRPC endpoint = %q, want %q", grpc, tt.wantGRPC)
			}
			http, err := instanceMetricsHTTPEndpoint(tt.raw)
			if err != nil {
				t.Fatal(err)
			}
			if http != tt.wantHTTP {
				t.Fatalf("HTTP endpoint = %q, want %q", http, tt.wantHTTP)
			}
		})
	}
}

func TestOTLPInstanceMetricsExporterDoesNothingWithoutExplicitEndpoint(t *testing.T) {
	t.Setenv("OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "")

	if err := (otlpInstanceMetricsExporter{}).Export(context.Background(), InstanceMetricsSnapshot{}); err != nil {
		t.Fatalf("export without an explicit endpoint: %v", err)
	}
}

func TestInstanceMetricDomainHandlesSchemeLessURL(t *testing.T) {
	if got := instanceMetricDomain("plane.example.test:8443/app"); got != "plane.example.test:8443" {
		t.Fatalf("domain = %q, want scheme-less host with port", got)
	}
	if got := instanceMetricDomain("https://plane.example.test/app"); got != "plane.example.test" {
		t.Fatalf("domain = %q, want host", got)
	}
}

func TestOTLPInstanceMetricsExporterSendsHTTPProtobuf(t *testing.T) {
	var gotMethod, gotPath, gotContentType string
	var gotBytes int
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		gotMethod, gotPath, gotContentType = request.Method, request.URL.Path, request.Header.Get("Content-Type")
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read OTLP payload: %v", err)
		}
		gotBytes = len(body)
		response.Header().Set("Content-Type", "application/x-protobuf")
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("OTLP_METRICS_PROTOCOL", "http")
	t.Setenv("OTLP_ENDPOINT", server.URL)

	err := (otlpInstanceMetricsExporter{}).Export(context.Background(), InstanceMetricsSnapshot{
		ServiceName: "plane-test",
		InstanceID:  "instance-test",
		Observations: []InstanceMetricObservation{{
			Name: "plane_instance_users_total", Description: "User count", Value: 7,
			Attributes: map[string]string{"instance_id": "instance-test"},
		}},
	})
	if err != nil {
		t.Fatalf("export OTLP metrics: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/metrics" || !strings.HasPrefix(gotContentType, "application/x-protobuf") || gotBytes == 0 {
		t.Fatalf("OTLP request method=%q path=%q content-type=%q bytes=%d", gotMethod, gotPath, gotContentType, gotBytes)
	}
}

func TestOTLPInstanceMetricsExporterIgnoresUnrelatedHTTPTraceEndpoint(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	firstRead := make(chan byte, 1)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer connection.Close()
		var firstByte [1]byte
		if _, readErr := io.ReadFull(connection, firstByte[:]); readErr == nil {
			firstRead <- firstByte[0]
		}
	}()
	t.Cleanup(func() {
		_ = listener.Close()
	})

	// The ordinary traces endpoint is HTTP. Metrics still use their configured
	// endpoint and default to TLS unless the metrics-specific flag says otherwise.
	t.Setenv("OTLP_METRICS_PROTOCOL", "grpc")
	t.Setenv("OTLP_ENDPOINT", "https://"+listener.Addr().String())
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://unrelated.example.test:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_INSECURE", "")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_CERTIFICATE", "")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_TIMEOUT", "1000")
	t.Setenv("OTEL_EXPORTER_OTLP_CERTIFICATE", "")
	t.Setenv("OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE", "")
	t.Setenv("OTEL_EXPORTER_OTLP_CLIENT_KEY", "")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_CLIENT_CERTIFICATE", "")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_CLIENT_KEY", "")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err = (otlpInstanceMetricsExporter{}).Export(ctx, InstanceMetricsSnapshot{
		ServiceName: "plane-test",
		InstanceID:  "instance-test",
		Observations: []InstanceMetricObservation{{
			Name: "plane_instance_users_total", Description: "User count", Value: 7,
			Attributes: map[string]string{"instance_id": "instance-test"},
		}},
	})
	if err == nil {
		t.Fatal("export through raw TCP endpoint unexpectedly succeeded")
	}
	select {
	case firstByte := <-firstRead:
		if firstByte != 0x16 {
			t.Fatalf("metrics transport first byte = %#02x, want TLS ClientHello (0x16)", firstByte)
		}
	default:
		t.Fatalf("metrics exporter did not connect to its configured endpoint: %v", err)
	}
}

func TestOTLPInstanceMetricsExporterPreservesMetricsCertificate(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	certificateServer := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	certificate := certificateServer.TLS.Certificates[0]
	certificateServer.Close()
	certPath := filepath.Join(t.TempDir(), "collector.pem")
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate.Certificate[0]}), 0o600); err != nil {
		t.Fatal(err)
	}

	var received atomic.Int32
	server := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{certificate},
		NextProtos:   []string{"h2"},
	})))
	colmetricpb.RegisterMetricsServiceServer(server, &recordingMetricsService{received: &received})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})
	t.Setenv("OTLP_METRICS_PROTOCOL", "grpc")
	t.Setenv("OTLP_ENDPOINT", "https://127.0.0.1:"+strconv.Itoa(listener.Addr().(*net.TCPAddr).Port))
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://unrelated.example.test:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_INSECURE", "")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_CERTIFICATE", certPath)
	t.Setenv("OTEL_EXPORTER_OTLP_CERTIFICATE", "")
	t.Setenv("OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE", "")
	t.Setenv("OTEL_EXPORTER_OTLP_CLIENT_KEY", "")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_CLIENT_CERTIFICATE", "")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_CLIENT_KEY", "")

	if err := (otlpInstanceMetricsExporter{}).Export(context.Background(), InstanceMetricsSnapshot{
		ServiceName: "plane-test",
		InstanceID:  "instance-test",
		Observations: []InstanceMetricObservation{{
			Name: "plane_instance_users_total", Description: "User count", Value: 7,
			Attributes: map[string]string{"instance_id": "instance-test"},
		}},
	}); err != nil {
		t.Fatalf("export metrics with configured certificate: %v", err)
	}
	if got := received.Load(); got == 0 {
		t.Fatal("configured TLS metrics endpoint received no metrics requests")
	}
}

func TestOTLPInstanceMetricsExporterHonorsMetricsInsecureSetting(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var received atomic.Int32
	server := grpc.NewServer()
	colmetricpb.RegisterMetricsServiceServer(server, &recordingMetricsService{received: &received})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})
	t.Setenv("OTLP_METRICS_PROTOCOL", "grpc")
	t.Setenv("OTLP_ENDPOINT", "http://"+listener.Addr().String())
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://unrelated.example.test:4317")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_INSECURE", "true")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_CERTIFICATE", "")
	t.Setenv("OTEL_EXPORTER_OTLP_CERTIFICATE", "")
	t.Setenv("OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE", "")
	t.Setenv("OTEL_EXPORTER_OTLP_CLIENT_KEY", "")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_CLIENT_CERTIFICATE", "")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_CLIENT_KEY", "")

	if err := (otlpInstanceMetricsExporter{}).Export(context.Background(), InstanceMetricsSnapshot{
		ServiceName: "plane-test",
		InstanceID:  "instance-test",
		Observations: []InstanceMetricObservation{{
			Name: "plane_instance_users_total", Description: "User count", Value: 7,
			Attributes: map[string]string{"instance_id": "instance-test"},
		}},
	}); err != nil {
		t.Fatalf("export metrics with explicit insecure setting: %v", err)
	}
	if got := received.Load(); got == 0 {
		t.Fatal("configured insecure metrics endpoint received no metrics requests")
	}
}

type recordingMetricsService struct {
	colmetricpb.UnimplementedMetricsServiceServer
	received *atomic.Int32
}

func (service *recordingMetricsService) Export(context.Context, *colmetricpb.ExportMetricsServiceRequest) (*colmetricpb.ExportMetricsServiceResponse, error) {
	service.received.Add(1)
	return &colmetricpb.ExportMetricsServiceResponse{}, nil
}
