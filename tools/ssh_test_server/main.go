package main

import (
	srv "arbor-exfil/tools/sshserv"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	stop, err := srv.Start("127.0.0.1:20222")
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "failed to start test ssh server:", err)
		os.Exit(1)
	}
	_, _ = fmt.Fprintln(os.Stderr, "test ssh server listening on 127.0.0.1:20222")
	defer stop()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
