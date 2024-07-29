package zaploki

import (
	"testing"

	"github.com/grafana/loki/pkg/push"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type TestClient struct {
	entries chan push.Entry
}

func (t *TestClient) Chan() chan<- push.Entry {
	return t.entries
}

func (t *TestClient) Stop() {
	close(t.entries)
}

var (
	_ LokiClient[push.Entry] = (*TestClient)(nil)
)

var logHandler LogHandler = NewHandler[push.Entry](&TestClient{}, SinkConfig{})

func TestNewCoreWithCreateLogger(t *testing.T) {
	type args struct {
		lc        LogHandler
		zapConfig zap.Config
	}
	tests := []struct {
		name string
		args args
		want *zap.Logger
	}{
		{
			name: "Test core",
			args: args{
				lc:        logHandler,
				zapConfig: zap.NewDevelopmentConfig(),
			},
			want: &zap.Logger{},
		},
		{
			name: "Test core 2",
			args: args{
				lc:        logHandler,
				zapConfig: zap.NewProductionConfig(),
			},
			want: &zap.Logger{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCoreWithCreateLogger(tt.args.lc, tt.args.zapConfig)
			if !assert.NoError(t, err, "NewCoreWithCreateLogger() error = %v", err) {
				return
			}
			assert.IsType(t, tt.want, got)
		})
	}
}
