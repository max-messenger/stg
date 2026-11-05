package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"stg/pkg/telemetry"
)

const (
	writeStatsPeriod = 10 * time.Second
	unknownAddr      = "unknown"
)

const (
	metricLabelAddr   = "addr"
	metricLabelAction = "action"
	metricLabelError  = "error"
)

type BeforeConnectCallback func(ctx context.Context, cc *pgx.ConnConfig) error
type AfterConnectCallback func(ctx context.Context, c *pgx.Conn) error

type Queryable interface {
	Exec(
		ctx context.Context, sql string, arguments ...any,
	) (commandTag pgconn.CommandTag, err error)

	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row

	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

//TODO: https://github.com/exaring/otelpgx add

type Postgres struct {
	cfg    *Config
	logger *zap.Logger

	newBackoff func() backoff.BackOff

	beforeConnectCallbacks []BeforeConnectCallback
	afterConnectCallbacks  []AfterConnectCallback

	pool *pgxpool.Pool
	addr string
}

func NewPostgres(
	cfg *Config,
	logger *zap.Logger,

	beforeConnectCallbacks []BeforeConnectCallback,
	afterConnectCallbacks []AfterConnectCallback,
) *Postgres {
	pg := &Postgres{
		cfg:                    cfg,
		logger:                 logger,
		newBackoff:             cfg.Backoff.NewBackOff(),
		beforeConnectCallbacks: beforeConnectCallbacks,
		afterConnectCallbacks:  afterConnectCallbacks,
	}

	go pg.reportStats()

	return pg
}

func (p *Postgres) callRaw(
	ctx context.Context,
	callFunc func(ctx context.Context, db Queryable) error,
) (addr string, err error) {
	con, err := p.pool.Acquire(ctx)
	if err != nil {
		return unknownAddr, err
	}

	defer con.Release()

	addr = con.Conn().PgConn().Conn().RemoteAddr().String()

	return addr, callFunc(ctx, con.Conn())
}

func (p *Postgres) runInTxRaw(
	ctx context.Context,
	name string,
	callFunc func(ctx context.Context, tx Queryable) error,
	txOptions pgx.TxOptions,
) (addr string, err error) {
	con, err := p.pool.Acquire(ctx)
	if err != nil {
		return unknownAddr, err
	}

	defer con.Release()

	addr = con.Conn().PgConn().Conn().RemoteAddr().String()

	tx, err := con.Conn().BeginTx(ctx, txOptions)
	if err != nil {
		return addr, err
	}

	rollback := func() {
		if rErr := tx.Rollback(ctx); rErr != nil {
			p.logger.Error(
				"failed to rollback transaction", zap.Error(rErr), zap.String("name", name),
			)
		}
	}

	defer func() {
		if r := recover(); r != nil {
			p.logger.Error("panic", zap.Any("panic", r))

			rollback()

			switch x := r.(type) {
			case error:
				err = fmt.Errorf("panic recovered: %w", x)
			default:
				err = fmt.Errorf("panic recovered: %v", x)
			}
		}
	}()

	err = callFunc(ctx, tx)

	if err != nil {
		rollback()

		return addr, err
	}

	return addr, tx.Commit(ctx)
}

func (p *Postgres) call(
	ctx context.Context,
	name string,
	callFunc func(ctx context.Context, db Queryable) error,
) (err error) {
	addr := unknownAddr
	ctx, span := otel.Tracer("postgres").Start(ctx, "call", trace.WithAttributes(
		attribute.String("postgres.query", name),
		attribute.String("postgres.query.type", "call"),
	))
	defer span.End()

	defer func(ts time.Time) {
		duration.With(prometheus.Labels{
			metricLabelAddr:   addr,
			metricLabelAction: name,
			metricLabelError:  telemetry.ErrLabelValue(err),
		}).Observe(time.Since(ts).Seconds())

		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		}
	}(time.Now())

	addr, err = p.callRaw(ctx, callFunc)
	span.SetAttributes(attribute.String("postgres.addr", addr))

	return err
}

func (p *Postgres) runInTx(
	ctx context.Context,
	name string,
	callFunc func(ctx context.Context, tx Queryable) error,
	txOptions pgx.TxOptions,
) (err error) {
	addr := unknownAddr
	ctx, span := otel.Tracer("postgres").Start(ctx, "tx", trace.WithAttributes(
		attribute.String("postgres.query", name),
		attribute.String("postgres.query.type", "tx"),
	))
	defer span.End()

	defer func(ts time.Time) {
		duration.With(prometheus.Labels{
			metricLabelAddr:   addr,
			metricLabelAction: name,
			metricLabelError:  telemetry.ErrLabelValue(err),
		}).Observe(time.Since(ts).Seconds())

		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		}
	}(time.Now())

	addr, err = p.runInTxRaw(ctx, name, callFunc, txOptions)
	span.SetAttributes(attribute.String("postgres.addr", addr))

	return err
}

func (p *Postgres) Call(
	ctx context.Context,
	name string,
	callFunc func(ctx context.Context, db Queryable) error,
) (err error) {
	bo := p.newBackoff()
	if bo == nil {
		return p.call(ctx, name, callFunc)
	}

	addr := unknownAddr
	ctx, span := otel.Tracer("postgres").Start(ctx, "call_backoff", trace.WithAttributes(
		attribute.String("postgres.query", name),
		attribute.String("postgres.query.type", "call_backoff"),
	))
	defer span.End()

	tryNum := 0

	// Include backoff delays in one observation, attributed to the last attempt.
	defer func(ts time.Time) {
		duration.With(prometheus.Labels{
			metricLabelAddr:   addr,
			metricLabelAction: name,
			metricLabelError:  telemetry.ErrLabelValue(err),
		}).Observe(time.Since(ts).Seconds())

		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		}
	}(time.Now())

	err = backoff.RetryNotify(
		func() error {
			tryNum++

			var qerr error
			addr, qerr = p.callRaw(ctx, callFunc)
			span.SetAttributes(attribute.String("postgres.addr", addr))

			return qerr
		},
		backoff.WithContext(bo, ctx),
		func(err error, duration time.Duration) {
			p.logger.Warn(
				"postgres call failed, retrying",
				zap.String("name", name),
				zap.Int("try_num", tryNum),
				zap.Duration("retry_delay", duration),
				zap.Error(err),
			)
		})

	span.SetAttributes(attribute.Int("postgres.try_num", tryNum))

	return err
}

func (p *Postgres) RunInTx(
	ctx context.Context,
	name string,
	callFunc func(ctx context.Context, tx Queryable) error,
	txOptions pgx.TxOptions,
) (err error) {
	bo := p.newBackoff()
	if bo == nil {
		return p.runInTx(ctx, name, callFunc, txOptions)
	}

	addr := unknownAddr
	ctx, span := otel.Tracer("postgres").Start(ctx, "tx_backoff", trace.WithAttributes(
		attribute.String("postgres.query", name),
		attribute.String("postgres.query.type", "tx_backoff"),
	))
	defer span.End()

	// Include backoff delays in one observation, attributed to the last attempt.
	defer func(ts time.Time) {
		duration.With(prometheus.Labels{
			metricLabelAddr:   addr,
			metricLabelAction: name,
			metricLabelError:  telemetry.ErrLabelValue(err),
		}).Observe(time.Since(ts).Seconds())

		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		}
	}(time.Now())

	tryNum := 0

	err = backoff.RetryNotify(
		func() error {
			tryNum++

			var qerr error
			addr, qerr = p.runInTxRaw(ctx, name, callFunc, txOptions)
			span.SetAttributes(attribute.String("postgres.addr", addr))

			return qerr
		},
		backoff.WithContext(bo, ctx),
		func(err error, duration time.Duration) {
			p.logger.Warn(
				"postgres transaction failed, retrying",
				zap.String("name", name),
				zap.Int("try_num", tryNum),
				zap.Duration("retry_delay", duration),
				zap.Error(err),
			)
		})

	span.SetAttributes(attribute.Int("postgres.try_num", tryNum))

	return err
}

func (p *Postgres) Start(ctx context.Context) error {
	poolCfg, err := pgxpool.ParseConfig(p.cfg.DatabaseURL)
	if err != nil {
		return err
	}

	poolCfg.BeforeConnect = p.beforeConnect
	poolCfg.AfterConnect = p.afterConnect
	poolCfg.ConnConfig.Tracer = otelpgx.NewTracer()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return err
	}

	p.pool = pool
	p.addr = pool.Config().ConnConfig.Host

	return nil
}

func (p *Postgres) Stop(_ context.Context) error {
	p.pool.Close()

	return nil
}

func (p *Postgres) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

func (p *Postgres) beforeConnect(ctx context.Context, cc *pgx.ConnConfig) error {
	for _, callback := range p.beforeConnectCallbacks {
		if err := callback(ctx, cc); err != nil {
			return err
		}
	}

	return nil
}

func (p *Postgres) afterConnect(ctx context.Context, c *pgx.Conn) error {
	for _, callback := range p.afterConnectCallbacks {
		if err := callback(ctx, c); err != nil {
			return err
		}
	}

	return nil
}

func (p *Postgres) reportStats() {
	for range time.NewTicker(writeStatsPeriod).C {
		if p.pool == nil {
			return // pool is not started
		}
		stat := p.pool.Stat()
		connections.WithLabelValues(p.addr, "max").Set(float64(stat.MaxConns()))
		connections.WithLabelValues(p.addr, "acquired").Set(float64(stat.AcquiredConns()))
		connections.WithLabelValues(p.addr, "idle").Set(float64(stat.IdleConns()))
		connections.WithLabelValues(p.addr, "total").Set(float64(stat.TotalConns()))
	}
}

func (p *Postgres) RawPool() *pgxpool.Pool {
	return p.pool
}
