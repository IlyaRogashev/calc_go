package application

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"github.com/IlyaRogashev/calc_go/calc_servise/internal/calc" 
)

type Config struct {
	Addr                string
	TimeAddition        int
	TimeSubtraction     int
	TimeMultiplications int
	TimeDivisions       int
}

func ConfigFromEnv() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	ta, _ := strconv.Atoi(os.Getenv("TIME_ADDITION_MS"))
	if ta == 0 {
		ta = 100
	}
	ts, _ := strconv.Atoi(os.Getenv("TIME_SUBTRACTION_MS"))
	if ts == 0 {
		ts = 100
	}
	tm, _ := strconv.Atoi(os.Getenv("TIME_MULTIPLICATIONS_MS"))
	if tm == 0 {
		tm = 100
	}
	td, _ := strconv.Atoi(os.Getenv("TIME_DIVISIONS_MS"))
	if td == 0 {
		td = 100
	}
	return &Config{
		Addr:                port,
		TimeAddition:        ta,
		TimeSubtraction:     ts,
		TimeMultiplications: tm,
		TimeDivisions:       td,
	}
}

type Orchestrator struct {
	calc.UnimplementedCalcServer
	Config      *Config
	db          *sqlx.DB
	mu          sync.Mutex
	exprCounter int64
	taskCounter int64
}

func NewOrchestrator() *Orchestrator {
	db, err := sqlx.Connect("sqlite3", "calcgo.db")
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}
	schema := `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	login TEXT UNIQUE NOT NULL,
	password_hash TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS expressions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	expr TEXT NOT NULL,
	status TEXT NOT NULL,
	result REAL,
	FOREIGN KEY(user_id) REFERENCES users(id)
);
CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY,
  expr_id INTEGER NOT NULL,
  arg1 REAL,
  arg2 REAL,
  operation TEXT,
  operation_time INTEGER,
  in_progress BOOLEAN NOT NULL DEFAULT 0,
  done BOOLEAN NOT NULL DEFAULT 0,
  UNIQUE(expr_id, arg1, arg2, operation),
  FOREIGN KEY(expr_id) REFERENCES expressions(id)
);
`
	if _, err := db.Exec(schema); err != nil {
		log.Fatal("migrate failed:", err)
	}
	return &Orchestrator{Config: ConfigFromEnv(), db: db}
}

func (o *Orchestrator) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req struct{ Login, Password string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	hash, err := HashPassword(req.Password)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if _, err := o.db.Exec("INSERT INTO users(login,password_hash) VALUES(?,?)", req.Login, hash); err != nil {
		http.Error(w, "user exists", http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (o *Orchestrator) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct{ Login, Password string }
	json.NewDecoder(r.Body).Decode(&req)
	var id int
	var hash string
	err := o.db.Get(&hash, "SELECT password_hash FROM users WHERE login=?", req.Login)
	if err != nil {
		http.Error(w, "invalid creds", http.StatusUnauthorized)
		return
	}
	err = CheckPassword(hash, req.Password)
	if err != nil {
		http.Error(w, "invalid creds", http.StatusUnauthorized)
		return
	}
	o.db.Get(&id, "SELECT id FROM users WHERE login=?", req.Login)
	tok, err := CreateToken(id)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": tok})
}

func (o *Or
