package rediskv

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Client struct{ addr string }

func NewFromEnv() *Client {
	addr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if addr == "" {
		addr = "redis:6379"
	}
	if strings.EqualFold(os.Getenv("REDIS_ENABLED"), "false") {
		return nil
	}
	return &Client{addr: addr}
}

func (c *Client) Enabled() bool { return c != nil && c.addr != "" }

func (c *Client) do(ctx context.Context, args ...string) (string, error) {
	if !c.Enabled() {
		return "", fmt.Errorf("redis disabled")
	}
	d := net.Dialer{Timeout: 800 * time.Millisecond}
	conn, err := d.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(1200 * time.Millisecond))
	var b strings.Builder
	b.WriteString("*" + strconv.Itoa(len(args)) + "\r\n")
	for _, a := range args {
		b.WriteString("$" + strconv.Itoa(len(a)) + "\r\n" + a + "\r\n")
	}
	if _, err := conn.Write([]byte(b.String())); err != nil {
		return "", err
	}
	r := bufio.NewReader(conn)
	p, err := r.ReadByte()
	if err != nil {
		return "", err
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	switch p {
	case '+', ':':
		return line, nil
	case '-':
		return "", fmt.Errorf(line)
	case '$':
		n, _ := strconv.Atoi(line)
		if n < 0 {
			return "", nil
		}
		buf := make([]byte, n+2)
		if _, err := r.Read(buf); err != nil {
			return "", err
		}
		return string(buf[:n]), nil
	default:
		return "", fmt.Errorf("unsupported redis response: %q %q", string(p), line)
	}
}

func (c *Client) Get(ctx context.Context, key string) (string, bool, error) {
	v, err := c.do(ctx, "GET", key)
	if err != nil {
		if strings.Contains(err.Error(), "nil") {
			return "", false, nil
		}
		return "", false, err
	}
	if v == "" {
		return "", false, nil
	}
	return v, true, nil
}
func (c *Client) SetEX(ctx context.Context, key, val string, ttl time.Duration) error {
	_, err := c.do(ctx, "SET", key, val, "PX", strconv.FormatInt(ttl.Milliseconds(), 10))
	return err
}
func (c *Client) Del(ctx context.Context, key string) error {
	_, err := c.do(ctx, "DEL", key)
	return err
}
func (c *Client) Incr(ctx context.Context, key string) (int, error) {
	s, err := c.do(ctx, "INCR", key)
	if err != nil {
		return 0, err
	}
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n, nil
}
func (c *Client) Expire(ctx context.Context, key string, ttl time.Duration) error {
	_, err := c.do(ctx, "PEXPIRE", key, strconv.FormatInt(ttl.Milliseconds(), 10))
	return err
}
