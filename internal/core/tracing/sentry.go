package tracing

import (
	"fmt"

	"github.com/getsentry/sentry-go"
)

// InitSentry intialize sentry for trace monitoring
func InitSentry(env string, dsn string) error {
	err := sentry.Init(sentry.ClientOptions{
		Dsn: dsn,
		// Set TracesSampleRate to 1.0 to capture 100%
		// of transactions for performance monitoring.
		// We recommend adjusting this value in production,
		EnableTracing:      true,
		TracesSampleRate:   1.0,
		ProfilesSampleRate: 1.0,
		Environment:        env,
		AttachStacktrace:   true,
	})
	if err != nil {
		customErr := fmt.Errorf("sentry initialization failed: %v", err)
		return customErr
	}
	return nil
}
