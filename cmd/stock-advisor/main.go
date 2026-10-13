// Command stock-advisor reads a TCBS portfolio and reports on it. It is
// read-only: it never places orders or moves money.
package main

import (
	"context"
	"os"
	"os/signal"
	_ "time/tzdata" // CGO_ENABLED=0 builds may run where the system has no tz database
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := run(ctx, productionDeps(), os.Args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}
