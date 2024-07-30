package zaploki

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/loki/pkg/push"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const lokiSinkKey = "loki"

type LokiClient[Entry any] interface {
	Chan() chan<- Entry
	Stop()
}

type jsonLogEntry struct {
	Level      string  `json:"level"`
	Timestamp  float64 `json:"ts"`
	LoggerName string  `json:"logger"`
	Message    string  `json:"msg"`
	Caller     string  `json:"caller"`
	Stack      string  `json:"stacktrace"`
	raw        string
}

type LogLineBuilder func(zapEntry zapcore.Entry, logFields []zapcore.Field) string

type EntryLike interface {
	push.Entry
}

type Sink[T any] struct {
	LogHandler
	// Zap sink compatible
	zap.Sink
	client         LokiClient[T]
	printFieldKey  bool
	loglineBuilder LogLineBuilder
	dynamicLabels  map[string]string
	entryConverter func(e push.Entry) T
}

var (
	_ zap.Sink   = (*Sink[any])(nil)
	_ LogHandler = (*Sink[any])(nil)
)

// func NewSink(c LokiClient[push.Entry], cfg SinkConfig) LogHandler {
// 	sink := &Sink{
// 		client:         c,
// 		printFieldKey:  cfg.PrintFieldKey,
// 		loglineBuilder: cfg.LoglineBuilder,
// 		dynamicLabels:  make(map[string]string),
// 	}

// 	if cfg.LoglineBuilder == nil {
// 		cfg.LoglineBuilder = sink.defaultLineBuilder
// 	}

// 	for _, v := range cfg.DynamicLabels {
// 		sink.dynamicLabels[v] = ""
// 	}

// 	return sink
// }

func (s *Sink[T]) defaultConverter(e push.Entry) T {
	return any(e).(T)
}

func NewHandler[T any](c LokiClient[T], cfg SinkConfig) LogHandler {
	sink := &Sink[T]{
		client:         c,
		printFieldKey:  cfg.PrintFieldKey,
		loglineBuilder: cfg.LoglineBuilder,
		dynamicLabels:  make(map[string]string),
	}

	sink.entryConverter = sink.defaultConverter

	if cfg.LoglineBuilder == nil {
		sink.loglineBuilder = sink.defaultLineBuilder
	}

	for _, v := range cfg.DynamicLabels {
		sink.dynamicLabels[v] = ""
	}

	return sink
}

// func (s *Sink) ZapSinkDefaultHandler(_ *url.URL) (zap.Sink, error) {
// 	return s, nil
// }

// func (s *Sink) WithCreateLogger(cfg zap.Config) (*zap.Logger, error) {
// 	return cfg.Build(zap.Hooks(s.ZapHookDefaultHandler))
// }

// Hook is a function that can be used as a zap hook to write log lines to loki
// func (s *Sink) ZapHookDefaultHandler(e zapcore.Entry) error {
// 	return s.Proceed(e, []zapcore.Field{})
// }

/**
* Zap sink implementation
 */
func (s *Sink[T]) Sync() error {
	// s.client.Flush()
	return nil
}
func (s *Sink[T]) Close() error {
	s.client.Stop()
	return nil
}

// Backward compatible, should not use if necessary
func (s *Sink[T]) Write(p []byte) (int, error) {
	var entry jsonLogEntry
	err := json.Unmarshal(p, &entry)
	if err != nil {
		// If the log is not Json encoded, or the format of the json string has been change, let bundle it whole as a log line
		entry = jsonLogEntry{
			Level:     "info",
			Timestamp: float64(time.Now().UnixMilli()),
			Message:   string(p),
		}
	}
	entry.raw = string(p)

	// convert jsonLogEntry to logproto.Entry
	e := push.Entry{
		Timestamp: time.UnixMilli(int64(entry.Timestamp) * int64(time.Millisecond)),
		Line:      entry.Message,
	}
	s.client.Chan() <- s.entryConverter(e)
	return len(p), nil
}

func (s *Sink[T]) Proceed(zapEntry zapcore.Entry, fields []zapcore.Field) error {
	metadata, loglineFields := extractDynamicLabelsFromFields(s.dynamicLabels, fields)

	e := push.Entry{
		Timestamp:          zapEntry.Time,
		Line:               s.loglineBuilder(zapEntry, loglineFields),
		StructuredMetadata: metadata,
	}

	s.client.Chan() <- s.entryConverter(e)
	return nil
}

func FromFieldToString(f zapcore.Field) string {
	switch f.Type {
	case zapcore.BoolType:
		return strconv.FormatBool(f.Integer == 1)
	case zapcore.ByteStringType:
		return string(f.Interface.([]byte))
	case zapcore.Complex128Type:
		return strconv.FormatComplex(f.Interface.(complex128), 'f', -1, 128)
	case zapcore.Complex64Type:
		return strconv.FormatComplex(f.Interface.(complex128), 'f', -1, 64)
	case zapcore.DurationType:
		return strconv.FormatInt(int64(time.Duration(f.Integer)), 10)
	case zapcore.Float64Type:
		return strconv.FormatFloat(math.Float64frombits(uint64(f.Integer)), 'f', -1, 64)
	case zapcore.Float32Type:
		return strconv.FormatFloat(float64(math.Float32frombits(uint32(f.Integer))), 'f', -1, 32)
	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
		return strconv.FormatInt(f.Integer, 10)
	case zapcore.StringType:
		return f.String
	case zapcore.TimeType:
		var t time.Time
		if f.Interface != nil {
			t = time.Unix(0, f.Integer).In(f.Interface.(*time.Location))
		} else {
			t = time.Unix(0, f.Integer)
		}
		return t.Format(time.RFC3339)
	case zapcore.TimeFullType:
		return f.Interface.(time.Time).Format(time.RFC3339)
	case zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type:
		return strconv.FormatUint(uint64(f.Integer), 10)
	case zapcore.StringerType:
		return f.Interface.(fmt.Stringer).String()
	case zapcore.ErrorType:
		return f.Interface.(error).Error()
	case zapcore.SkipType:
		break
	default:
		return fmt.Sprintf("<unsupported field \"%v\" value>", f)
	}
	return ""
}

func (s *Sink[T]) defaultLineBuilder(zapEntry zapcore.Entry, logFields []zapcore.Field) string {
	var b strings.Builder
	separator := " "

	b.WriteString("level=")
	b.WriteString(zapEntry.Level.String())

	b.WriteString(separator)
	b.WriteString("caller=")
	b.WriteString(zapEntry.Level.String())

	b.WriteString(separator)
	b.WriteString(zapEntry.Message + convertFieldsToStr(logFields, s.printFieldKey))

	if zapEntry.Level == zapcore.ErrorLevel && zapEntry.Stack != "" {
		b.WriteString(separator)
		b.WriteString(zapEntry.Stack)
	}

	return b.String()
}

func convertFieldsToStr(fields []zapcore.Field, printFieldKey bool) string {
	var b strings.Builder
	separator := ", "

	for _, field := range fields {
		b.WriteString(separator)
		if printFieldKey {
			b.WriteString(field.Key)
			b.WriteString("=")
		}
		b.WriteString(FromFieldToString(field))
	}

	return b.String()
}

func extractDynamicLabelsFromFields(labels map[string]string, fields []zapcore.Field) (metadata push.LabelsAdapter, loglineFields []zapcore.Field) {
	metadata = push.LabelsAdapter{}
	loglineFields = []zapcore.Field{}
	for _, field := range fields {
		if defaultVal, ok := labels[field.Key]; ok {
			l := push.LabelAdapter{
				Name:  field.Key,
				Value: FromFieldToString(field),
			}
			if l.Value == "" {
				l.Value = defaultVal
			}

			metadata = append(metadata, l)
		} else {
			loglineFields = append(loglineFields, field)
		}
	}
	return metadata, loglineFields
}
