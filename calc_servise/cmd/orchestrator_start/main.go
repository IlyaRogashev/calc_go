package main

import (
    "flag"
    "log"
    "net/http"
    "os"
    "time"
)

func main() {
    log.SetFlags(log.LstdFlags | log.Lmicroseconds)
    flag.Parse()

    orchestrator := &orchestrator.DefaultOrchestrator{}
    orchestrator.Start()

    simpleAgent := &agent.SimpleAgent{}
    orchestrator.AddAgent(simpleAgent)

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello World!"))
    })

    log.Fatal(http.ListenAndServe(":8080", nil))

    orchestrator.Stop()
}
