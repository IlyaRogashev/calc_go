package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    orchestrator := orchestrator.NewOrchestrator(ctx)

    router := mux.NewRouter()

    orchestrator.RegisterHandlers(router, orchestrator)

    srv := &http.Server{
        Addr:    ":8000",
        Handler: router,
    }

    go orchestrator.Start()

    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("listen: %s\n", err)
        }
    }()

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    log.Println("Shutting down server...")

    orchestrator.Stop()

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("server shutdown failed: %v", err)
    }
    log.Println("Server exited properly")
}
