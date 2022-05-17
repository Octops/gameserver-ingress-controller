package runtime

import (
	"context"
	"os"
	"os/signal"
)

func SetupSignal(ctx context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(ctx, os.Interrupt)
}
