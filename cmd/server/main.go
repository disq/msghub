package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/disq/msghub"
)

func main() {
	addr := flag.String("addr", fmt.Sprintf(":%v", msghub.DefaultPort), "Listen on address")
	flag.Parse()

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	logger := log.New(os.Stderr, "", log.LstdFlags|log.LUTC)
	s := NewServer(ctx, logger)

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM, syscall.SIGPIPE)
		<-ch
		logger.Print("Got signal, cleaning up...")
		cancelFunc()
	}()

	err := s.Listen(*addr)
	if err != nil && err != context.Canceled && !isErrNetClosing(err) {
		log.Print(err)
	}
}
