package response

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"

	"github.com/labstack/echo/v4"
	"github.com/mdobak/go-xerrors"
)

type GeneralResponse struct {
	Status  string `json:"status"`
	Data    any    `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

type StackFrame struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`
}

func ResponseSuccess(
	ctx echo.Context,
	code int,
	data any,
) error {
	if data == nil {
		return ctx.JSON(code, GeneralResponse{
			Status:  "success",
			Message: "success",
		})
	}

	if reflect.ValueOf(data).Type().Kind() == reflect.String {
		return ctx.JSON(code, GeneralResponse{
			Status:  "success",
			Message: data.(string),
		})
	}

	return ctx.JSON(code, GeneralResponse{
		Status: "success",
		Data:   data,
	})
}

func NewHttpError(
	code int,
	internal error,
	message ...any,
) *echo.HTTPError {
	if len(message) == 0 && internal != nil {
		message = append(message, internal.Error())
	}

	rs := echo.NewHTTPError(code, message...)
	if internal != nil {
		rs.Internal = xerrors.New(internal)
	}

	return rs
}

func ResponseError(
	ctx echo.Context,
	err error,
) error {
	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		httpErr = NewHttpError(http.StatusInternalServerError, err, "Internal server error.")
	}

	status := "error"
	if httpErr.Code >= http.StatusBadRequest && httpErr.Code < http.StatusInternalServerError {
		status = "fail"
	}

	var message, errMsg string
	if reflect.ValueOf(httpErr.Message).Type().Kind() == reflect.String {
		message = httpErr.Message.(string)
	}

	if httpErr.Internal != nil {
		errMsg = httpErr.Internal.Error()
		if message == "" {
			message = errMsg
		}
	}

	if TraceHelperInstance != nil {
		traceId, ok := ctx.Get("traceId").(string)
		if ok {
			reqInfo := ExtractRequestData(ctx, httpErr.Internal)
			if err := TraceHelperInstance.Save(traceId, reqInfo); err != nil {
				slog.Error("error when saving trace data", slog.Any("err", err))
			}
		}

	}

	if httpErr.Internal != nil && status == "error" {
		slog.ErrorContext(
			ctx.Request().Context(),
			httpErr.Internal.Error(),
			slog.Any("data", httpErr.Internal),
			slog.Any("message", httpErr.Message),
			slog.Any("err", errMsg),
		)
	} else {
		slog.ErrorContext(
			ctx.Request().Context(),
			fmt.Sprint(httpErr.Message),
			slog.Any("data", "logic error"),
			slog.Any("message", httpErr.Message),
			slog.Any("err", errMsg),
		)
	}

	return ctx.JSON(httpErr.Code, GeneralResponse{
		Status:  status,
		Message: message,
	})
}

func ResponseFailMessage(
	ctx echo.Context,
	code int,
	message string,
) error {
	status := "fail"
	if code >= 400 && code < 500 {
		status = "fail"
	}
	return ctx.JSON(code, GeneralResponse{
		Status:  status,
		Message: message,
	})
}

type RequestInfo struct {
	Url     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Ip      string            `json:"ip"`
	Query   map[string]string `json:"query"`
	Params  map[string]string `json:"params"`
	Body    string            `json:"body"`
	Stack   []StackFrame      `json:"stack"`
}

var ExcludedHeaders = map[string]bool{
	"Authorization": true,
}

func ExtractRequestData(ctx echo.Context, err error) *RequestInfo {
	req := ctx.Request()
	reqInfo := RequestInfo{
		Url:    req.URL.String(),
		Method: req.Method,
		Ip:     ctx.RealIP(),
	}

	headers := make(map[string]string)
	for k, v := range req.Header {
		if ExcludedHeaders[k] {
			continue
		}

		headers[k] = v[0]
	}
	reqInfo.Headers = headers

	query := make(map[string]string)
	for k, v := range req.URL.Query() {
		query[k] = v[0]
	}
	reqInfo.Query = query

	params := make(map[string]string)
	for i, v := range ctx.ParamValues() {
		params[ctx.ParamNames()[i]] = v
	}
	reqInfo.Params = params

	body := ctx.Request().Body
	if body == nil {
		limitReader := io.LimitReader(body, 1024)
		buf, err := io.ReadAll(limitReader)
		if err == nil {
			reqInfo.Body = string(buf)
		}
	}

	trace := xerrors.StackTrace(err)
	if len(trace) != 0 {
		for i := 2; i < len(trace.Frames())-3; i++ {
			reqInfo.Stack = append(reqInfo.Stack, StackFrame{
				File:     trace.Frames()[i].File,
				Line:     trace.Frames()[i].Line,
				Function: trace.Frames()[i].Function,
			})
		}
	}

	return &reqInfo
}
