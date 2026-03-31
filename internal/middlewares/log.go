package middlewares

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	custom_log "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/log"
)

func AddLogContext() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			traceId := ctx.Request().Header.Get(models.StreamTraceIdHeader)
			if traceId == "" {
				traceId = uuid.New().String()
			}

			data := []slog.Attr{
				slog.Any("trace_id", traceId),
				slog.Any("ip", ctx.RealIP()),
			}
			ctx.SetRequest(ctx.Request().WithContext(context.WithValue(
				ctx.Request().Context(),
				custom_log.SlogFieldsKey,
				data,
			)))

			ctx.Set("traceId", traceId)

			return next(ctx)
		}
	}
}

func AddGrpcLogContext(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	requestId := uuid.New()
	start := time.Now()
	data := []slog.Attr{
		slog.Any("request_id", requestId),
		slog.Any("method", info.FullMethod),
	}

	if req != nil {
		data = append(data, slog.Any("request", req))
	}

	ctx = context.WithValue(ctx, custom_log.SlogFieldsKey, data)
	resp, err := handler(ctx, req)
	if err != nil {
		var msg string
		st, ok := status.FromError(err)
		if ok {
			msg = st.Message()
		} else {
			msg = err.Error()
		}

		slog.Error(
			"gRPC Fail req",
			slog.Any("response-time", convertLatency(time.Since(start))),
			slog.Any("err", err),
			slog.Any("msg", msg),
		)
		return nil, err
	} else {
		slog.InfoContext(ctx,
			"gRPC OK req",
			slog.Any("response-time", convertLatency(time.Since(start))),
			slog.Any("response", resp),
		)
	}

	return resp, nil
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the wrapper's context instead of the server stream's context
func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

func AddGrpcLogContextStream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		requestId := uuid.New()
		start := time.Now()

		// Logging when the stream starts
		data := []slog.Attr{
			slog.Any("request_id", requestId),
			slog.Any("method", info.FullMethod),
			slog.Bool("is_client_stream", info.IsClientStream),
			slog.Bool("is_server_stream", info.IsServerStream),
		}

		// We're not capturing the request body as it's a stream
		ctx := stream.Context()
		ctx = context.WithValue(ctx, custom_log.SlogFieldsKey, data)

		// Create a wrapped stream that has our enhanced context
		wrappedStream := &wrappedServerStream{
			ServerStream: stream,
			ctx:          ctx,
		}

		slog.InfoContext(ctx, "gRPC stream started")

		// Handle the RPC
		err := handler(srv, wrappedStream)

		// Log the result
		if err != nil {
			var msg string
			st, ok := status.FromError(err)
			if ok {
				msg = st.Message()
			} else {
				msg = err.Error()
			}
			slog.ErrorContext(ctx,
				"gRPC stream failed",
				slog.Any("response-time", convertLatency(time.Since(start))),
				slog.Any("err", err),
				slog.Any("msg", msg),
			)
		} else {
			slog.InfoContext(ctx,
				"gRPC stream completed successfully",
				slog.Any("response-time", convertLatency(time.Since(start))),
			)
		}

		return err
	}
}

func convertLatency(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%d ns", d.Nanoseconds())
	} else if d < time.Millisecond {
		return fmt.Sprintf("%.2f µs", float64(d.Microseconds()))
	} else if d < time.Second {
		return fmt.Sprintf("%.2f ms", float64(d.Milliseconds()))
	}

	return fmt.Sprintf("%.2f s", d.Seconds())
}
