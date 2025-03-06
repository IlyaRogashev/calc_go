package main

import (
    "bytes"
    "encoding/json"
    "io/ioutil"
    "log"
    "net/http"
    "time"
)
  
func main() {
    agent := &agent.SimpleAgent{}

    for {
        task, err := getTask()
        if err != nil {
            log.Printf("Error getting task: %v\n", err)
            time.Sleep(5 * time.Second) // Ждем перед следующей попыткой
            continue
        }
        result, err := agent.Calculate(task.Expression)
        if err != nil {
            log.Printf("Error calculating expression '%s': %v\n", task.Expression, err)
            continue
        }
        err = submitResult(task.ID, result)
        if err != nil {
            log.Printf("Error submitting result: %v\n", err)
        }

        // Пауза перед следующим запросом
        time.Sleep(1 * time.Second)
    }
}

func getTask() (*agent.Task, error) {
    client := &http.Client{}
    req, _ := http.NewRequest("GET", "http://localhost:8000/internal/task", nil)
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    var task agent.Task
    err = json.Unmarshal(body, &task)
    if err != nil {
        return nil, err
    }

    return &task, nil
}

func submitResult(id string, result float64) error {
    client := &http.Client{}
    data := struct {
        ID     string  `json:"id"`
        Result float64 `json:"result"`
    }{
        ID:     id,
        Result: result,
    }

    payload, err := json.Marshal(data)
    if err != nil {
        return err
    }

    req, _ := http.NewRequest("PUT", "http://localhost:8000/internal/task", bytes.NewBuffer(payload))
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
