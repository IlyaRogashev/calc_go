package main

import (
	"log"
	"os"

	"google.golang.org/grpc"
	"github.com/IlyaRogashev/calc_go/calc_servise/application"
)

func main() {
	orchestrator := application.NewOrchestrator()
	go func() {
		if err := orchestrator.RunServer(); err != nil {
			log.Fatalf("failed to run orchestrator: %v", err)
		}
	}()

	agent := application.NewAgent()
	agent.Run()

	select {}
}
