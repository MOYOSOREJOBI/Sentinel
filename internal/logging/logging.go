package logging

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

type Logger struct{ service string }

func New(service string) Logger { return Logger{service: service} }

func (l Logger) Info(msg string, fields map[string]any)  { l.out("info", msg, fields) }
func (l Logger) Error(msg string, fields map[string]any) { l.out("error", msg, fields) }

func (l Logger) out(level, msg string, fields map[string]any) {
	m := map[string]any{"ts": time.Now().UTC().Format(time.RFC3339Nano), "level": level, "service": l.service, "msg": msg}
	for k, v := range fields {
		m[k] = v
	}
	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(m); err != nil {
		log.Println(level, msg)
	}
}
