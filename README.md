# HTTP Servers in Go

## TABLE OF CONTENTS

- [What I learned](#what-i-learned)
  - [File (Pull) vs. Network (Push) I/O](#file-pull-vs-network-push-io)
  - [HTTP/1.1 RFCs & Protocol Evolution](#http11-rfcs--protocol-evolution)
  - [HTTP Message Wire Format](#http-message-wire-format)
    - [The `request-line`](#the-request-line)
    - [The `status-line`](#the-status-line)
    - [Response Headers & Early Routing](#response-headers--early-routing)
    - [Why `CRLF`?](#why-crlf)
  - [Stream Boundaries & Pipelining](#stream-boundaries--pipelining)
  - [Chunked Streaming & HTTP Trailers](#chunked-streaming--http-trailers)
  - [Concurrency: `sync/atomic` vs. `sync.Mutex`](#concurrency-syncatomic-vs-syncmutex)
  - [Buffer Allocation (`len` vs. `cap`)](#buffer-allocation-len-vs-cap)
  - [Debugging & Testing](#debugging--testing)

## What I learned

### File (Pull) vs. Network (Push) I/O

- with files ( pull ), you are in control of the timing, chunk size, and when to stop reading since the data is local and static

- with networks ( push ), the remote sender dictates the arrival rate, chunk size, and termination so you just have to react as bytes come in

- both use the exact same `io.Reader` interface in Go, which means network reads will give you unpredictable byte counts (`0 < n <= len(p)`) and will block until bytes actually arrive, the connection closes, or a deadline hits

### HTTP/1.1 RFCs & Protocol Evolution

- **RFC 9110** covers the modern core HTTP semantics like methods, status codes, and headers ( [RFC 9110](https://datatracker.ietf.org/doc/html/rfc9110) — supersedes RFC 7231 / RFC 2616 )

- **RFC 9112** is specifically for the HTTP/1.1 wire format and syntax specs ( [RFC 9112](https://datatracker.ietf.org/doc/html/rfc9112) )

```text
[HTTP/1.1] vs [HTTP/2] vs [HTTP/3]

HTTP/1.1 (Text over TCP):
- text-based delimited framing (CRLF, Content-Length, chunked hex markers)
- suffers from Head-of-Line (HoL) blocking at the application level; pipelining requires responses to come back in the exact same order

HTTP/2 (Binary Framing over TCP):
- replaces text parsing with binary frames (HEADERS, DATA, etc.) which gets rid of syntax ambiguities and parsing overhead, though you lose human-readable debugging with raw netcat/telnet
- true multiplexing: multiple request/response streams interleave over a single TCP connection without blocking each other
- header compression (HPACK) using static/dynamic tables + Huffman coding to save bandwidth on repetitive headers
- server push allows sending assets (like CSS/JS) before the client even asks for them

HTTP/3 (Binary Framing over QUIC/UDP):
- drops TCP for QUIC over UDP, which fixes TCP-level HoL blocking (a dropped packet on stream A won't stall stream B) and speeds up handshakes (0-RTT / 1-RTT)
- TLS 1.3 encryption is baked directly into the QUIC transport layer instead of being slapped on top

```

### HTTP Message Wire Format

Every HTTP/1.1 message is strictly made of **four parts**, delimited by `CRLF` (`\r\n` — easy way to remember the order: *Registered Nurse*)

```http
start-line CRLF
*( field-line CRLF )
CRLF
[ message-body ]

```

* `start-line` identifies what's happening. For requests it's strictly called the `request-line`, and for responses it's the `status-line`
* `*( field-line CRLF )` is zero or more key-value header pairs (`Key: Value\r\n`)
* `CRLF` is a mandatory empty line that signals the end of the headers block
* `[ message-body ]` is the optional raw payload ( its size is bounded by `Content-Length` or `Transfer-Encoding: chunked` )

#### The `request-line`

> `request-line` (ABNF Grammar per RFC 9112)
> `request-line = method SP request-target SP HTTP-version`
> Wire example: `GET /coffee HTTP/1.1`

* `method` is a case-sensitive token for the action (`GET`, `POST`, `PUT`, `DELETE`)
* `SP` is just a single space character delimiter ( ASCII 32 )
* `request-target` is the URI/resource path being acted upon ( e.g. `/coffee` )
* `HTTP-version` is the exact protocol declaration (`HTTP/1.1`, following `HTTP-name "/" DIGIT "." DIGIT`)

#### The `status-line`

> `status-line` (ABNF Grammar per RFC 9112)
> `status-line = HTTP-version SP status-code SP [ reason-phrase ]`

* consists of the `HTTP-version` (`HTTP/1.1`), a 3-digit `status-code` (`200`, `404`), and an optional `reason-phrase` (`OK`, `Not Found`)
* weird wire quirk: the separating space (`SP`) after `status-code` is **mandatory** even if you leave out the `reason-phrase` ( so `HTTP/1.1 200 \r\n` is valid, `HTTP/1.1 200\r\n` is not )
* clients should ignore the `reason-phrase` anyway since it's unreliable across proxies/locales and was completely dropped in HTTP/2 and HTTP/3

#### Response Headers & Early Routing

* while there isn't an official list of required response headers, you almost always want these three:
* `Content-Length` for the payload size in bytes ( needed for framing )
* `Connection` to control whether the socket stays open (`keep-alive` vs `close`)
* `Content-Type` for the payload MIME type ( e.g. `text/plain`, `application/json` )


* the `request-line` alone actually has everything you need to route a request (`method` + `request-target`)
* high-performance parsers can dispatch to the route handler right after reading line 1, and frameworks that do this let you `await` ( or block on a channel for ) the rest of the headers only if the handler actually needs them

#### Why `CRLF`?

* goes back to old mechanical teletypes that needed two physical movements: `\r` (Carriage Return) slammed the print head back to column 0, and `\n` (Line Feed) rolled the paper down one line
* if you only send `\n` without post-processing, the cursor drops straight down from where it is and gives you that classic "staircase" effect
* Unix/macOS abstract this by treating a bare `\n` as both, while Windows kept the explicit `\r\n`
* the HTTP specs went with `\r\n` so raw byte streams could be piped directly to serial terminals and hardware printers without needing the OS to translate anything

### Stream Boundaries & Pipelining

* TCP is just a continuous byte stream with zero concept of "messages", so HTTP has to build its own framing on top of that raw stream

```text
How do you know when a message ends?

1. Content-Length (Fixed): server calculates the payload size upfront, client stops reading once it hits that exact byte count.
2. Transfer-Encoding: chunked (Streaming): sends the size per chunk and finishes with a zero-chunk (0\r\n\r\n). Lets you keep the TCP connection alive (keep-alive) without having to buffer the whole response in RAM first.
3. Connection Teardown (EOF fallback): if both headers are missing, you just read until the TCP socket closes. Kills connection reuse and makes curl -v complain.

```

* any bytes sitting past `Content-Length` **cannot** just be ignored—they belong to the next pipelined HTTP request on that same socket ( "leave for later", don't discard )

```text
POST /a HTTP/1.1\r\nContent-Length: 5\r\n\r\nhelloGET /b HTTP/1.1\r\n...
└──────────── request A ────────────┘└──── request B ────...

```

* real production parsers slice the buffer up to `remaining` bytes (`data[:remaining]`) and keep the leftover bytes in the reader buffer for the next request cycle ( in the course we just returned `ErrInvalidBody` when `len(data) > remaining` to keep state management simple )

### Chunked Streaming & HTTP Trailers

* chunking solves the streaming dilemma: since TCP is just a pipe, dynamic responses without chunking would force you to either buffer the whole thing in memory just to get a `Content-Length`, or kill the TCP connection when you're done
* in HTTP/1.1, trailers and chunked framing **only** work with chunked encoding because `Content-Length` messages have no delimiter to separate the body from trailing headers ( HTTP/2 and HTTP/3 support trailers natively using `HEADERS` frames )

> The mandatory `0\r\n\r\n` terminator:
> * the first `\r\n` closes the zero-size chunk (`0\r\n`)
> * the second `\r\n` closes the `trailer-section` (`*( field-line CRLF ) CRLF`)
> 
> 
> Even if you don't send a `Trailer` header and have zero trailers, that second `\r\n` is **always** required. Forgetting it leaves the parser hanging on the socket waiting for the trailer section to close, which causes client hangs or read timeouts.

* wire layout when trailers are included:
```http
0\r\n
Trailer-Key: Value\r\n
\r\n

```


* if you're sending trailers, you have to pre-announce the keys upfront in the main headers so the parser knows what to allocate for:
```http
Transfer-Encoding: chunked
Trailer: Content-Digest, Server-Timing

```


* trailer values obviously can't be resolved until the consumer finishes reading the body all the way to `io.EOF`
* standard trailers include things like `Content-Digest` ( [RFC 9530](https://datatracker.ietf.org/doc/html/rfc9530) ) or `Server-Timing`. Custom ones like `X-Checksum` work too, though the `X-` prefix convention was officially deprecated in [RFC 6648](https://datatracker.ietf.org/doc/html/rfc6648)
* framing and routing headers are **banned** in trailers ( you can't put `Content-Length`, `Transfer-Encoding`, `Host`, or `Authorization` at the end of a stream )

### Concurrency: `sync/atomic` vs. `sync.Mutex`

* `sync.Mutex` starts out fast as a single atomic CPU instruction, but under contention it escalates to spin-locking and eventually falls back to OS thread parking (`futex` / `pthread_mutex`), which triggers expensive context switches
* `atomic.Bool` ( and `atomic.Int64` / `atomic.Pointer[T]` ) is a hardware-level single CPU instruction—lock-free, never blocks, and handles CPU cache coherency directly
* use `atomic` for single, isolated state flags or counters, and reach for `sync.Mutex` when you have multiple state mutations that need to happen together as one atomic unit
* clean server shutdown pattern using `atomic.Bool` to ignore `net.ErrClosed` when we intentionally stop the listener:
```go
type Server struct {
    listener net.Listener
    closed   atomic.Bool
}

func (s *Server) Close() error {
    s.closed.Store(true)
    return s.listener.Close()
}

func (s *Server) listen() {
    for {
        conn, err := s.listener.Accept()
        if err != nil {
            if s.closed.Load() {
                return // we closed it on purpose, ignore the error
            }
            continue
        }
        go s.handle(conn)
    }
}

```



### Buffer Allocation (`len` vs. `cap`)

* `make([]byte, 8)` allocates length 8 and capacity 8. It's zero-initialized and immediately indexable (`buf[0] = 1`), which is what you need when passing a destination buffer directly into `r.Read(buf)` ( `Read` only fills up to `len(buf)`, so a length of 0 would read 0 bytes! )
* `make([]byte, 0, 8)` allocates length 0 and capacity 8. Useful when you want to pre-allocate the underlying array so `append()` doesn't have to keep reallocating and copying memory

### Debugging & Testing

* you can pipe raw socket reads directly into `tee` to inspect the exact HTTP byte stream coming over the wire without touching your server code:
```bash
# terminal 1: run listener and dump raw bytes to a file
go run ./cmd/tcplistener | tee /tmp/rawget.http

# terminal 2: fire off a request
curl http://localhost:42069/coffee

```


* Go's standard `testing` package is super imperative and verbose with all the manual `if got != want { t.Errorf(...) }` checks
* [stretchr/testify](https://github.com/stretchr/testify) gives you declarative assertions (`assert.Equal(t, want, got)`, `require.NoError(t, err)`) that feel much closer to PHPUnit while still running natively with `go test`

|  | `sync/atomic` | `sync.Mutex` |
| --- | --- | --- |
| mechanism | single CPU instruction | atomic fast-path -> OS thread park |
| blocking | lock-free (never) | blocks under contention |
| best for | single flags / counters | multi-variable state mutations |
