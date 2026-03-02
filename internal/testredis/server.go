package testredis

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type item struct {
	v   string
	exp time.Time
}

type Server struct {
	ln   net.Listener
	mu   sync.Mutex
	data map[string]item
}

func Start() (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &Server{ln: ln, data: map[string]item{}}
	go s.serve()
	return s, nil
}

func (s *Server) Addr() string { return s.ln.Addr().String() }
func (s *Server) Close() error { return s.ln.Close() }

func (s *Server) serve() {
	for {
		c, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(c)
	}
}

func (s *Server) handle(c net.Conn) {
	defer c.Close()
	r := bufio.NewReader(c)
	for {
		args, err := readRESPArray(r)
		if err != nil {
			return
		}
		if len(args) == 0 {
			_, _ = c.Write([]byte("-ERR empty\r\n"))
			continue
		}
		cmd := strings.ToUpper(args[0])
		s.mu.Lock()
		now := time.Now()
		for k, v := range s.data {
			if !v.exp.IsZero() && now.After(v.exp) {
				delete(s.data, k)
			}
		}
		s.mu.Unlock()
		switch cmd {
		case "GET":
			if len(args) != 2 {
				_, _ = c.Write([]byte("-ERR\r\n"))
				continue
			}
			s.mu.Lock()
			v, ok := s.data[args[1]]
			s.mu.Unlock()
			if !ok {
				_, _ = c.Write([]byte("$-1\r\n"))
				continue
			}
			_, _ = c.Write([]byte(fmt.Sprintf("$%d\r\n%s\r\n", len(v.v), v.v)))
		case "SET":
			if len(args) < 3 {
				_, _ = c.Write([]byte("-ERR\r\n"))
				continue
			}
			it := item{v: args[2]}
			if len(args) >= 5 && strings.ToUpper(args[3]) == "PX" {
				ms, _ := strconv.Atoi(args[4])
				it.exp = time.Now().Add(time.Duration(ms) * time.Millisecond)
			}
			s.mu.Lock()
			s.data[args[1]] = it
			s.mu.Unlock()
			_, _ = c.Write([]byte("+OK\r\n"))
		case "DEL":
			if len(args) != 2 {
				_, _ = c.Write([]byte("-ERR\r\n"))
				continue
			}
			s.mu.Lock()
			delete(s.data, args[1])
			s.mu.Unlock()
			_, _ = c.Write([]byte(":1\r\n"))
		case "INCR":
			if len(args) != 2 {
				_, _ = c.Write([]byte("-ERR\r\n"))
				continue
			}
			s.mu.Lock()
			it := s.data[args[1]]
			n, _ := strconv.Atoi(it.v)
			n++
			it.v = strconv.Itoa(n)
			s.data[args[1]] = it
			s.mu.Unlock()
			_, _ = c.Write([]byte(":" + strconv.Itoa(n) + "\r\n"))
		case "PEXPIRE":
			if len(args) != 3 {
				_, _ = c.Write([]byte("-ERR\r\n"))
				continue
			}
			ms, _ := strconv.Atoi(args[2])
			s.mu.Lock()
			it, ok := s.data[args[1]]
			if ok {
				it.exp = time.Now().Add(time.Duration(ms) * time.Millisecond)
				s.data[args[1]] = it
			}
			s.mu.Unlock()
			if ok {
				_, _ = c.Write([]byte(":1\r\n"))
			} else {
				_, _ = c.Write([]byte(":0\r\n"))
			}
		default:
			_, _ = c.Write([]byte("-ERR unknown\r\n"))
		}
	}
}

func readRESPArray(r *bufio.Reader) ([]string, error) {
	p, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	if p != '*' {
		return nil, fmt.Errorf("expected array")
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	n, _ := strconv.Atoi(strings.TrimSpace(line))
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		if p, _ = r.ReadByte(); p != '$' {
			return nil, fmt.Errorf("expected bulk")
		}
		line, err = r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		l, _ := strconv.Atoi(strings.TrimSpace(line))
		buf := make([]byte, l+2)
		if _, err := r.Read(buf); err != nil {
			return nil, err
		}
		out = append(out, string(buf[:l]))
	}
	return out, nil
}
