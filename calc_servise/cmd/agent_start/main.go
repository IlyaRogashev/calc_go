package main

import (
    "context"
    "log"
    "os"
    "strconv"
    "time"

)

func main() {
    serverURL := os.Getenv("SERVER_URL")
    if serverURL == "" {
        log.Fatalf("SERVER_URL environment variable is not set")
    }

    computingPowerStr := os.Getenv("COMPUTING_POWER")
    if computingPowerStr == "" {
        log.Fatalf("COMPUTING_POWER environment variable is not set")
    }

    computingPower, err := strconv.Atoi(computingPowerStr)
    if err != nil {
        log.Fatalf("Failed to parse COMPUTING_POWER: %v", err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    agent := agent.NewAgent(ctx, computingPower, serverURL)

    agent.Start()

    log.Println("Press Ctrl+C to stop the agent.")
    select {}
}
