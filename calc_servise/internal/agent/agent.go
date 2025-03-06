package agent

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "strconv"
    "sync"
    "time"
)

type Agent struct {
    ctx    context.Context
    cancel context.CancelFunc
    wg     sync.WaitGroup
    mux    sync.RWMutex
    // Каналы для управления работой агента
    newTasksChan chan Task
    resultsChan  chan Result
    stopChan     chan struct{}
    // Параметры конфигурации
    computingPower int
    serverURL     string
}

type Task struct {
    ID           string
    Expression   string
    Operation    string
    OperationTime time.Time
}

type Result struct {
    ID     string
    Result float64
}

func NewAgent(ctx context.Context, computingPower int, serverURL string) *Agent {
    ctx, cancel := context.WithCancel(ctx)

    return &Agent{
        ctx:             ctx,
        cancel:          cancel,
        computingPower:  computingPower,
        serverURL:       serverURL,
        newTasksChan:    make(chan Task),
        resultsChan:     make(chan Result),
        stopChan:        make(chan struct{}),
    }
}

func (a *Agent) Start() {
    a.wg.Add(a.computingPower)
    for i := 0; i < a.computingPower; i++ {
        go a.startWorker(i)
    }

    go a.taskManager()
    go a.resultSender()

    log.Printf("Agent started with %d workers", a.computingPower)
}

func (a *Agent) Stop() {
    a.cancel()
    close(a.stopChan)
    a.wg.Wait()
    log.Println("Agent stopped")
}

func (a *Agent) startWorker(index int) {
    defer a.wg.Done()

    log.Printf("Worker #%d started", index)

    for {
        select {
        case task := <-a.newTasksChan:
            log.Printf("Worker #%d received task: %v", index, task)
            result, err := calc.Calc(task.Expression)
            if err != nil {
                log.Printf("Worker #%d failed to compute task: %v, error: %v", index, task, err)
                continue
            }

            a.resultsChan <- Result{ID: task.ID, Result: result}
            log.Printf("Worker #%d computed task: %v, result: %f", index, task, result)
        case <-a.ctx.Done():
            log.Printf("Worker #%d stopping...", index)
            return
        }
    }
}

func (a *Agent) taskManager() {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            task, err := a.getTask()
            if err != nil {
                log.Printf("Error getting task: %v", err)
                continue
            }

            if task != nil {
                a.mux.RLock()
                a.newTasksChan <- *task
                a.mux.RUnlock()
            }
        case <-a.ctx.Done():
            log.Println("Task manager stopping...")
            return
        }
    }
}

func (a *Agent) resultSender() {
    for {
        select {
        case result := <-a.resultsChan:
            err := a.submitResult(result)
            if err != nil {
                log.Printf("Error submitting result: %v", err)
            }
        case <-a.ctx.Done():
            log.Println("Result sender stopping...")
            return
        }
    }
}

func (a *Agent) getTask() (*Task, error) {
    client := &http.Client{}
    req, err := http.NewRequestWithContext(a.ctx, "GET", a.serverURL+"/internal/task", nil)
    if err != nil {
        return nil, err
    }

    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("non-200 response: %d", resp.StatusCode)
    }

    var task Task
    err = json.NewDecoder(resp.Body).Decode(&task)
    if err != nil {
        return nil, err
    }

    return &task, nil
}

func (a *Agent) submitResult(result Result) error {
    client := &http.Client{}
    payload, err := json.Marshal(result)
    if err != nil {
        return err
    }

    req, err := http.NewRequestWithContext(a.ctx, "PUT", a.serverURL+"/internal/task", bytes.NewBuffer(payload))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("non-200 response: %d", resp.StatusCode)
    }

    return nil
}
