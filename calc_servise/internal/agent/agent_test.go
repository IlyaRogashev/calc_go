package agent

import (
    "context"
    "testing"
    "time"
)

func TestAgent_Start(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    agent := NewAgent(ctx, 2, "http://example.com")

    agent.Start()

    time.Sleep(100 * time.Millisecond)

    require.True(t, agent.IsRunning())

    agent.Stop()

    require.False(t, agent.IsRunning())
}

func TestAgent_GetTask(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    agent := NewAgent(ctx, 2, "http://example.com")

    task := Task{
        ID:         "123",
        Expression: "2+2",
    }

    agent.newTasksChan <- task
    time.Sleep(100 * time.Millisecond)

    select {
    case result := <-agent.resultsChan:
        require.Equal(t, "123", result.ID)
        require.Equal(t, 4.0, result.Result)
    default:
        t.FailNow()
    }
}

func TestAgent_SubmitResult(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    agent := NewAgent(ctx, 2, "http://example.com")

    result := Result{
        ID:     "123",
        Result: 4.0,
    }

    err := agent.submitResult(result)

    require.NoError(t, err)
}
