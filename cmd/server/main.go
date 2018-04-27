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
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM, syscall.SIGPIPE)
		<-ch
		log.Print("Got signal, cleaning up...")
		cancelFunc()
	}()

	logger := log.New(os.Stderr, "", log.LstdFlags|log.LUTC)

	s := NewServer(ctx, logger)

	go func() {
		// Shut down server on interrupt
		<-s.ctx.Done()
		s.Close()
	}()

	err := s.Listen(*addr)
	if err != nil && err != context.Canceled {
		log.Print(err)
	}

	// Make sure goroutines are down
	s.Close()
}
