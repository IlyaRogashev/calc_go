package orchestrator

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "sync"
    "time"

)

type Orchestrator struct {
    ctx              context.Context
    cancel           context.CancelFunc
    wg               sync.WaitGroup
    mux              sync.RWMutex
    agents           []*agent.Agent
    newTasksChan     chan Task
    resultsChan      chan Result
    stopChan         chan struct{}
    expressionStates map[string]*ExpressionState
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

type ExpressionState struct {
    ID        string
    Status    string
    Result    float64
    StartedAt time.Time
    EndedAt   time.Time
}

func NewOrchestrator(ctx context.Context, agents []*agent.Agent) *Orchestrator {
    ctx, cancel := context.WithCancel(ctx)

    return &Orchestrator{
        ctx:              ctx,
        cancel:           cancel,
        agents:           agents,
        newTasksChan:     make(chan Task),
        resultsChan:      make(chan Result),
        stopChan:         make(chan struct{}),
        expressionStates: make(map[string]*ExpressionState),
    }
}

func (o *Orchestrator) Start() {
    o.wg.Add(len(o.agents))
    for _, agent := range o.agents {
        go o.startAgent(agent)
    }

    go o.taskDistributor()
    go o.resultCollector()

    log.Println("Orchestrator started")
}

func (o *Orchestrator) Stop() {
    o.cancel()
    close(o.stopChan)
    o.wg.Wait()
    log.Println("Orchestrator stopped")
}

func (o *Orchestrator) startAgent(agent *agent.Agent) {
    defer o.wg.Done()

    log.Printf("Starting agent %p", agent)

    for {
        select {
        case task := <-o.newTasksChan:
            log.Printf("Assigning task %v to agent %p", task, agent)
            agent.ProcessTask(task)
        case <-o.ctx.Done():
            log.Printf("Stopping agent %p", agent)
            return
        }
    }
}

func (o *Orchestrator) taskDistributor() {
    for {
        select {
        case task := <-o.newTasksChan:
            o.assignTaskToAgent(task)
        case <-o.ctx.Done():
            log.Println("Task distributor stopping...")
            return
        }
    }
}

func (o *Orchestrator) assignTaskToAgent(task Task) {
    o.mux.RLock()
    defer o.mux.RUnlock()
    freeAgent := o.findFreeAgent()
    if freeAgent == nil {
        log.Printf("No free agents available for task %v", task)
        return
    }

    // Назначение задачи агенту
    freeAgent.ProcessTask(task)

    o.expressionStates[task.ID] = &ExpressionState{
        ID:     task.ID,
        Status: "assigned",
    }
}

func (o *Orchestrator) findFreeAgent() *agent.Agent {
    for _, agent := range o.agents {
        if agent.IsAvailable() {
            return agent
        }
    }
    return nil
}

func (o *Orchestrator) resultCollector() {
    for {
        select {
        case result := <-o.resultsChan:
            log.Printf("Received result for task %v: %f", result.ID, result.Result)
            o.updateExpressionState(result)
        case <-o.ctx.Done():
            log.Println("Result collector stopping...")
            return
        }
    }
}

func (o *Orchestrator) updateExpressionState(result Result) {
    o.mux.Lock()
    defer o.mux.Unlock()

    state, exists := o.expressionStates[result.ID]
    if !exists {
        log.Printf("Unknown task ID: %v", result.ID)
        return
    }

    state.Status = "completed"
    state.Result = result.Result
    state.EndedAt = time.Now()
}

func (o *Orchestrator) HandleNewTask(w http.ResponseWriter, r *http.Request) {
    decoder := json.NewDecoder(r.Body)
    var req struct {
        Expression string `json:"expression"`
    }
    err := decoder.Decode(&req)
    if err != nil {
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }

    taskID := fmt.Sprintf("%d", time.Now().UnixNano())

    task := Task{
        ID:         taskID,
        Expression: req.Expression,
    }

    o.newTasksChan <- task

    response := struct {
        ID string `json:"id"`
    }{
        ID: taskID,
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(response)
}

func (o *Orchestrator) HandleGetTaskResults(w http.ResponseWriter, r *http.Request) {
    o.mux.RLock()
    defer o.mux.RUnlock()

    states := make([]*ExpressionState, 0, len(o.expressionStates))
    for _, state := range o.expressionStates {
        states = append(states, state)
    }

    response := struct {
        States []*ExpressionState `json:"states"`
    }{
        States: states,
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(response)
}
