package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"

	"stg/pkg/connections/postgres"
)

func TestUnitPostgresOperationAddress(t *testing.T) {
	for _, transaction := range []bool{false, true} {
		t.Run(fmt.Sprintf("transaction=%t", transaction), func(t *testing.T) {
			for _, scenario := range []string{
				"success", "callback_error", "acquire_error", "retry_success", "retry_error", "retry_acquire_error",
				"begin_error", "panic",
			} {
				if !transaction && (scenario == "begin_error" || scenario == "panic") {
					continue
				}
				t.Run(scenario, func(t *testing.T) {
					exporter := &spanExporter{}
					previousProvider := otel.GetTracerProvider()
					provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
					otel.SetTracerProvider(provider)
					t.Cleanup(func() {
						otel.SetTracerProvider(previousProvider)
						require.NoError(t, provider.Shutdown(context.Background()))
					})

					operationErr := errors.New("operation failed")
					pg := newTestPostgres(t, scenario, operationErr)
					attempts := 0
					callback := func(ctx context.Context, db postgres.Queryable) error {
						attempts++
						if scenario == "panic" {
							panic(operationErr)
						}
						if strings.HasPrefix(scenario, "retry_") && attempts == 1 {
							// Force the next attempt to acquire a connection to a different endpoint.
							switch conn := db.(type) {
							case *pgx.Conn:
								require.NoError(t, conn.Close(ctx))
							case pgx.Tx:
								require.NoError(t, conn.Conn().Close(ctx))
							}
							return operationErr
						}
						if scenario == "callback_error" || scenario == "retry_error" {
							return operationErr
						}
						return nil
					}

					var err error
					spanName := "call"
					if transaction {
						spanName = "tx"
						err = pg.RunInTx(t.Context(), t.Name(), callback, pgx.TxOptions{})
					} else {
						err = pg.Call(t.Context(), t.Name(), callback)
					}
					switch scenario {
					case "success", "retry_success":
						require.NoError(t, err)
					case "begin_error":
						require.ErrorContains(t, err, "begin failed")
					default:
						require.ErrorIs(t, err, operationErr)
					}

					addr := "192.0.2.1:5432"
					if strings.HasPrefix(scenario, "retry_") {
						addr = "192.0.2.2:5432"
						spanName += "_backoff"
					}
					if scenario == "acquire_error" || scenario == "retry_acquire_error" {
						addr = "unknown"
					}
					assertDurationAddress(t, t.Name(), addr)
					exporter.mu.Lock()
					defer exporter.mu.Unlock()
					for _, span := range exporter.spans {
						if span.Name() == spanName {
							for _, attr := range span.Attributes() {
								if attr.Key == "postgres.addr" {
									require.Equal(t, addr, attr.Value.AsString())
									return
								}
							}
							t.Fatal("operation span has no postgres.addr attribute")
						}
					}
					t.Fatalf("operation span %q not found", spanName)
				})
			}
		})
	}
}

func newTestPostgres(t *testing.T, scenario string, operationErr error) *postgres.Postgres {
	t.Helper()
	var connects atomic.Int64
	beforeConnect := func(_ context.Context, cfg *pgx.ConnConfig) error {
		connect := connects.Add(1)
		if scenario == "acquire_error" || (scenario == "retry_acquire_error" && connect > 1) {
			return operationErr
		}
		cfg.DialFunc = func(_ context.Context, _, _ string) (net.Conn, error) {
			client, backend := net.Pipe()
			done := make(chan struct{})
			go func() {
				defer close(done)
				defer backend.Close()
				servePostgres(backend, scenario == "begin_error")
			}()
			t.Cleanup(func() {
				_ = client.Close()
				_ = backend.Close()
				<-done
			})
			return &addressedConn{
				Conn: client,
				addr: &net.TCPAddr{IP: net.ParseIP(fmt.Sprintf("192.0.2.%d", connect)), Port: 5432},
			}, nil
		}
		return nil
	}
	interval := time.Millisecond
	maxRetries := uint64(1)
	pg := postgres.NewPostgres(&postgres.Config{
		DatabaseURL: "postgres://user:password@localhost/database?sslmode=disable&pool_max_conns=1",
		Backoff: postgres.BackoffConfig{
			Enabled:         strings.HasPrefix(scenario, "retry_"),
			InitialInterval: &interval,
			MaxRetries:      &maxRetries,
		},
	}, zap.NewNop(), []postgres.BeforeConnectCallback{beforeConnect}, nil)
	require.NoError(t, pg.Start(t.Context()))
	t.Cleanup(func() { require.NoError(t, pg.Stop(context.Background())) })
	return pg
}

type addressedConn struct {
	net.Conn
	addr net.Addr
}

func (c *addressedConn) RemoteAddr() net.Addr {
	return c.addr
}

// servePostgres emulates only the wire-protocol messages needed to connect and
// begin/end a transaction. Tests use net.Pipe, with no database or TCP listener.
func servePostgres(conn net.Conn, failBegin bool) {
	backend := pgproto3.NewBackend(conn, conn)
	if _, err := backend.ReceiveStartupMessage(); err != nil {
		return
	}
	backend.Send(&pgproto3.AuthenticationOk{})
	backend.Send(&pgproto3.ParameterStatus{Name: "server_version", Value: "16.0"})
	backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
	if err := backend.Flush(); err != nil {
		return
	}
	for {
		message, err := backend.Receive()
		if err != nil {
			return
		}
		switch message := message.(type) {
		case *pgproto3.Query:
			status := byte('I')
			if strings.HasPrefix(message.String, "begin") {
				if failBegin {
					backend.Send(&pgproto3.ErrorResponse{Severity: "ERROR", Code: "XX000", Message: "begin failed"})
				} else {
					status = 'T'
					backend.Send(&pgproto3.CommandComplete{CommandTag: []byte("BEGIN")})
				}
			} else {
				backend.Send(&pgproto3.CommandComplete{CommandTag: []byte(strings.ToUpper(message.String))})
			}
			backend.Send(&pgproto3.ReadyForQuery{TxStatus: status})
			if err := backend.Flush(); err != nil {
				return
			}
		default:
			return
		}
	}
}

func assertDurationAddress(t *testing.T, operation, addr string) {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	var observations uint64
	for _, family := range families {
		if family.GetName() != "postgres_request_duration_seconds" {
			continue
		}
		for _, metric := range family.GetMetric() {
			labels := make(map[string]string)
			for _, label := range metric.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			if labels["action"] == operation {
				require.Equal(t, addr, labels["addr"])
				observations += metric.GetHistogram().GetSampleCount()
			}
		}
	}
	// Retries still produce one observation for the complete operation.
	require.Equal(t, uint64(1), observations)
}

type spanExporter struct {
	mu    sync.Mutex
	spans []sdktrace.ReadOnlySpan
}

func (e *spanExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spans = append(e.spans, spans...)
	return nil
}

func (*spanExporter) Shutdown(context.Context) error {
	return nil
}
