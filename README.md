# goon

Go on...

---

goon is a small utility script for intercepting network traffic and redirecting it to a single destination.

... No, it isn't particularly useful.

## Install

```bash
go get -u github.com/meanguy/goon
```

## Usage

Forward traffic all HTTP traffic to another host

```bash
$ goon -verbose -r google.com:80 &
[2] 1362764

$ curl -L -m5 localhost:8080
2021/04/23 01:26:57  info 127.0.0.1:8080 <- 127.0.0.1:49308 component=worker len=78 src=127.0.0.1:49308 workerId=0
2021/04/23 01:26:57  info 127.0.0.1:8080 -> 142.250.69.206:80 component=worker len=78 src=127.0.0.1:49308 workerId=0
curl: (28) Operation timed out after 5001 milliseconds with 0 bytes received
2021/04/23 01:27:02  info client connection closed  component=worker src=127.0.0.1:49308 workerId=0
```

## TODO

- Bidirectional traffic
- TCP payload inspection (ASCII, hex, base64 encodings)
