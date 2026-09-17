// Package fakebroker is a NATS server just complete enough to test a client
// against, spoken by hand over a unix socket.
//
// It exists so the SDK's tests can watch a real CONNECT be refused, and a real
// request be answered, without depending on nats-server. That matters twice
// over: nats-server is a heavy dependency for a module every plugin author
// imports, and — more importantly — this package contains no nats-io import at
// all, so the import-boundary test stays true with the test suite in place.
//
// It implements INFO, CONNECT, PING/PONG, SUB, UNSUB, PUB, HPUB and MSG, with
// real wildcard routing. It implements nothing else, and it is not a substrate:
// it is a scripted peer.
package fakebroker

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestCreds is a syntactically valid .creds file with a throwaway user key.
//
// It is a fixture, not a secret: nothing signs anything a real server would
// accept, and the fake broker below never checks it. What it has to be is
// PARSEABLE, because nats.go reads the seed to sign the server's nonce and
// fails before CONNECT if it cannot — which would make a test of a refused
// CONNECT pass for the wrong reason.
const TestCreds = `-----BEGIN NATS USER JWT-----
eyJ0eXAiOiJKV1QiLCJhbGciOiJlZDI1NTE5LW5rZXkifQ.eyJqdGkiOiJGQUtFIiwibmFtZSI6InRlc3QifQ.fake
------END NATS USER JWT------

************************* IMPORTANT *************************
NKEY Seed printed below can be used to sign and prove identity.
NKEYs are sensitive and should be treated as secrets.

-----BEGIN USER NKEY SEED-----
SUACC2CA432WTWJMZ3ZAE7QASHCLMALR2US5QKPXQGISIAK5UPWDZ2GE7M
------END USER NKEY SEED------

*************************************************************
`

// WriteCreds writes TestCreds into a temporary file and returns its path.
func WriteCreds(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bus.creds")
	if err := os.WriteFile(path, []byte(TestCreds), 0o400); err != nil {
		t.Fatal(err)
	}
	return path
}

// Responder answers one request. Returning nil sends no reply at all, which is
// how a test produces a timeout.
type Responder func(subject string, headers map[string]string, data []byte) []byte

// Broker is the fake server.
type Broker struct {
	// Socket is the unix socket path to dial.
	Socket string

	ln net.Listener

	mu sync.Mutex
	// refuse, when set, is the -ERR text sent instead of PONG at CONNECT.
	refuse     string
	responders map[string]Responder
	// violateOn is the subject a PUB on which draws a permissions violation.
	violateOn string
	conns     []*conn
}

// Start listens on a unix socket inside the test's temp dir and stops with the
// test.
//
// The socket path is kept short deliberately: a unix socket path is limited to
// 108 bytes, and a long temp dir turns the bind failure into what looks like a
// connect timeout.
func Start(t *testing.T) *Broker {
	t.Helper()
	dir, err := os.MkdirTemp("", "fb")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	path := filepath.Join(dir, "b.sock")
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatalf("listen %s: %v", path, err)
	}
	b := &Broker{Socket: path, ln: ln, responders: map[string]Responder{}}
	t.Cleanup(b.Close)
	go b.accept()
	return b
}

// Refuse makes the next CONNECT fail with text, e.g. "Authorization Violation".
func (b *Broker) Refuse(text string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refuse = text
}

// Respond registers an answer for one subject.
func (b *Broker) Respond(subject string, r Responder) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.responders[subject] = r
}

// ViolateOn makes a PUB or SUB naming subject draw an asynchronous
// permissions violation, the way a real server refuses an ungranted subject.
func (b *Broker) ViolateOn(subject string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.violateOn = subject
}

func (b *Broker) Close() {
	_ = b.ln.Close()
	b.mu.Lock()
	conns := b.conns
	b.conns = nil
	b.mu.Unlock()
	for _, c := range conns {
		_ = c.nc.Close()
	}
}

func (b *Broker) accept() {
	for {
		nc, err := b.ln.Accept()
		if err != nil {
			return
		}
		c := &conn{b: b, nc: nc, r: bufio.NewReader(nc), w: bufio.NewWriter(nc), subs: map[string]sub{}}
		b.mu.Lock()
		b.conns = append(b.conns, c)
		b.mu.Unlock()
		go c.serve()
	}
}

type sub struct {
	subject string
	queue   string
}

type conn struct {
	b  *Broker
	nc net.Conn
	r  *bufio.Reader
	w  *bufio.Writer

	mu   sync.Mutex
	subs map[string]sub
}

const info = `INFO {"server_id":"fake","server_name":"fake","version":"2.14.0","proto":1,"go":"go1.25","host":"0.0.0.0","port":4222,"headers":true,"auth_required":true,"max_payload":1048576,"client_id":1,"nonce":"aaaaaaaaaaaaaaaaaaaaaa"}`

func (c *conn) serve() {
	defer c.nc.Close()
	c.send(info)
	for {
		line, err := c.r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue
		}
		verb, rest, _ := strings.Cut(line, " ")
		switch strings.ToUpper(verb) {
		case "CONNECT":
			// The CONNECT is followed immediately by a PING, and the client
			// treats the next line as the answer to both.
			if text := c.refusal(); text != "" {
				c.send("-ERR '" + text + "'")
				// Do NOT close here. The client writes CONNECT and PING as two
				// writes, so a server that hangs up on the first one gives the
				// client a broken pipe on the second and it never reads the
				// -ERR at all — which turns a refused credential into a
				// transport error, intermittently. A real server closes after
				// the client has read; draining until EOF reproduces that
				// without a sleep.
				c.drain()
				return
			}
		case "PING":
			c.send("PONG")
		case "PONG":
		case "SUB":
			c.handleSub(rest)
		case "UNSUB":
			sid, _, _ := strings.Cut(rest, " ")
			c.mu.Lock()
			delete(c.subs, sid)
			c.mu.Unlock()
		case "PUB":
			c.handlePub(rest, false)
		case "HPUB":
			c.handlePub(rest, true)
		}
	}
}

// drain reads and discards until the peer closes, so the refusal it was just
// sent is the last thing it sees rather than a racing hangup.
func (c *conn) drain() {
	_ = c.nc.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 512)
	for {
		if _, err := c.nc.Read(buf); err != nil {
			return
		}
	}
}

func (c *conn) refusal() string {
	c.b.mu.Lock()
	defer c.b.mu.Unlock()
	return c.b.refuse
}

func (c *conn) handleSub(rest string) {
	f := strings.Fields(rest)
	var s sub
	var sid string
	switch len(f) {
	case 2:
		s.subject, sid = f[0], f[1]
	case 3:
		s.subject, s.queue, sid = f[0], f[1], f[2]
	default:
		return
	}
	if c.violates(s.subject) {
		c.send(fmt.Sprintf("-ERR 'Permissions Violation for Subscription to %q'", s.subject))
		return
	}
	c.mu.Lock()
	c.subs[sid] = s
	c.mu.Unlock()
}

// handlePub reads a PUB or HPUB and routes it.
func (c *conn) handlePub(rest string, withHeaders bool) {
	f := strings.Fields(rest)
	var subject, reply string
	var hdrLen, total int
	switch {
	case withHeaders && len(f) == 3:
		subject, hdrLen, total = f[0], atoi(f[1]), atoi(f[2])
	case withHeaders && len(f) == 4:
		subject, reply, hdrLen, total = f[0], f[1], atoi(f[2]), atoi(f[3])
	case !withHeaders && len(f) == 2:
		subject, total = f[0], atoi(f[1])
	case !withHeaders && len(f) == 3:
		subject, reply, total = f[0], f[1], atoi(f[2])
	default:
		return
	}
	buf := make([]byte, total+2) // payload + CRLF
	if _, err := readFull(c.r, buf); err != nil {
		return
	}
	body := buf[:total]
	headers := map[string]string{}
	if withHeaders && hdrLen > 0 && hdrLen <= len(body) {
		headers = parseHeaders(string(body[:hdrLen]))
		body = body[hdrLen:]
	}
	if c.violates(subject) {
		c.send(fmt.Sprintf("-ERR 'Permissions Violation for Publish to %q'", subject))
		return
	}
	if reply != "" {
		if r := c.responder(subject); r != nil {
			if out := r(subject, headers, body); out != nil {
				c.deliver(reply, out)
			}
			return
		}
	}
	c.route(subject, reply, body)
}

func (c *conn) violates(subject string) bool {
	c.b.mu.Lock()
	defer c.b.mu.Unlock()
	return c.b.violateOn != "" && c.b.violateOn == subject
}

func (c *conn) responder(subject string) Responder {
	c.b.mu.Lock()
	defer c.b.mu.Unlock()
	return c.b.responders[subject]
}

// route delivers to every matching subscription on every connection, which is
// what a broker does and what a per-connection loopback would silently get
// wrong the first time a test used two clients.
func (c *conn) route(subject, reply string, body []byte) {
	c.b.mu.Lock()
	conns := append([]*conn(nil), c.b.conns...)
	c.b.mu.Unlock()
	for _, other := range conns {
		other.mu.Lock()
		for sid, s := range other.subs {
			if !matches(s.subject, subject) {
				continue
			}
			other.msg(subject, sid, reply, body)
		}
		other.mu.Unlock()
	}
}

// deliver answers a request on the connection that made it.
func (c *conn) deliver(reply string, body []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for sid, s := range c.subs {
		if matches(s.subject, reply) {
			c.msg(reply, sid, "", body)
			return
		}
	}
}

func (c *conn) msg(subject, sid, reply string, body []byte) {
	head := "MSG " + subject + " " + sid + " "
	if reply != "" {
		head += reply + " "
	}
	head += strconv.Itoa(len(body))
	c.writeLocked(head, body)
}

// writeLocked writes one MSG. The caller already holds c.mu — msg is reached
// while routing across connections, which takes each target's lock itself.
func (c *conn) writeLocked(head string, body []byte) {
	_, _ = c.w.WriteString(head + "\r\n")
	_, _ = c.w.Write(body)
	_, _ = c.w.WriteString("\r\n")
	_ = c.w.Flush()
}

func (c *conn) send(line string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, _ = c.w.WriteString(line + "\r\n")
	_ = c.w.Flush()
}

// matches is NATS subject matching: "*" is one token, ">" is the rest.
func matches(pattern, subject string) bool {
	p := strings.Split(pattern, ".")
	s := strings.Split(subject, ".")
	for i, tok := range p {
		if tok == ">" {
			return i <= len(s)-1
		}
		if i >= len(s) {
			return false
		}
		if tok != "*" && tok != s[i] {
			return false
		}
	}
	return len(p) == len(s)
}

func parseHeaders(raw string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(raw, "\r\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		out[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}
	return out
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func readFull(r *bufio.Reader, buf []byte) (int, error) {
	n := 0
	for n < len(buf) {
		m, err := r.Read(buf[n:])
		n += m
		if err != nil {
			return n, err
		}
	}
	return n, nil
}
