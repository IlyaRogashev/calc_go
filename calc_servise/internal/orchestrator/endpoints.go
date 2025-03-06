package orchestrator

import (
    "encoding/json"
    "fmt"
    "log"
    "math/rand"
    "net/http"
    "sync"
    "time"
)

type Orchestrator interface {
    Start()
    Stop()
    AddAgent(agent Agent)
    ProcessExpression(expression string) (float64, error)
}

type DefaultOrchestrator struct {
    agents []Agent
    mu     sync.Mutex
    stopCh chan struct{}
    doneCh chan struct{}

    expressions map[string]*ExpressionStatus
    idGenerator *rand.Rand
}

type ExpressionStatus struct {
    ID          string
    Status      string
    Result      float64
    Expression  string
    CreatedAt   time.Time
    LastUpdated time.Time
}

const (
    Pending   = "pending"
    InProgress = "in_progress"
    Completed  = "completed"
    Failed     = "failed"
)

func (o *DefaultOrchestrator) Start() {
    o.stopCh = make(chan struct{})
    o.doneCh = make(chan struct{})
    o.expressions = make(map[string]*ExpressionStatus)
    o.idGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))

    go func() {
        defer close(o.doneCh)
        for {
            select {
            case <-o.stopCh:
                return
            default:
            }
        }
    }()
}

func (o *DefaultOrchestrator) Stop() {
    close(o.stopCh)
    <-o.doneCh
}

func (o *DefaultOrchestrator) AddAgent(agent Agent) {
    o.mu.Lock()
    defer o.mu.Unlock()
    o.agents = append(o.agents, agent)
}

func (o *DefaultOrchestrator) ProcessExpression(expression string) (float64, error) {
    o.mu.Lock()
    defer o.mu.Unlock()

    if len(o.agents) == 0 {
        return 0, errors.New("no agents available")
    }

    agent := o.agents[0]
    return agent.Calculate(expression)
}

func (o *DefaultOrchestrator) HandleAddCalculation(w http.ResponseWriter, r *http.Request) {
    decoder := json.NewDecoder(r.Body)
    var req struct {
        Expression string `json:"expression"`
    }
    err := decoder.Decode(&req)
    if err != nil {
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }

    id := fmt.Sprintf("%d", o.idGenerator.Int63())

    o.expressions[id] = &ExpressionStatus{
        ID:         id,
        Status:     Pending,
        Expression: req.Expression,
        CreatedAt:  time.Now(),
    }

    resp := struct {
        ID string `json:"id"`
    }{
        ID: id,
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(resp)
}
func (o *DefaultOrchestrator) HandleGetExpressions(w http.ResponseWriter, r *http.Request) {
    exprList := make([]*ExpressionStatus, 0, len(o.expressions))
    for _, expr := range o.expressions {
        exprList = append(exprList, expr)
    }

    // Отправляет ответ
    resp := struct {
        Expressions []*ExpressionStatus `json:"expressions"`
    }{
        Expressions: exprList,
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(resp)
}

func (o *DefaultOrchestrator) HandleGetExpressionByID(w http.ResponseWriter, r *http.Request) {
    params := mux.Vars(r)
    id := params["id"]

    expr, ok := o.expressions[id]
    if !ok {
        http.NotFound(w, r)
        return
    }

    resp := struct {
        Expression *ExpressionStatus `json:"expression"`
    }{
        Expression: expr,
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(resp)
}

func (o *DefaultOrchestrator) HandleGetTask(w http.ResponseWriter, r *http.Request) {
    var task *ExpressionStatus
    for _, expr := range o.expressions {
        if expr.Status == Pending {
            task = expr
            break
        }
    }

    if task == nil {
        http.NotFound(w, r)
        return
    }

    task.Status = InProgress
    task.LastUpdated = time.Now()

    resp := struct {
        Task *ExpressionStatus `json:"task"`
    }{
        Task: task,
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(resp)
}
