package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"net"
	"sync"
	"time"

	"github.com/meanguy/goon/lib/log"
	"github.com/meanguy/goon/lib/queue"
)

const timeToSleep = 250 * time.Millisecond

type args struct {
	Debug       bool
	Quiet       bool
	Verbose     bool
	LocalAddr   string
	RemoteAddr  string
	BufferSize  int
	WorkerCount int
	Timeout     time.Duration
}

func parseArgs() *args {
	opts := &args{}

	flag.BoolVar(&opts.Debug, "debug", false, "set log level to debug. Has precedence over -quiet and -verbose.")
	flag.BoolVar(&opts.Quiet, "quiet", false, "set log level to error.")
	flag.BoolVar(&opts.Verbose, "verbose", false, "set log level to info. Has precedence over -quiet.")
	flag.StringVar(&opts.LocalAddr, "l", "127.0.0.1:8080", "address for binding the local server socket")
	flag.StringVar(&opts.RemoteAddr, "r", "127.0.0.1:8000", "address for forwarding socket data to")
	flag.IntVar(&opts.BufferSize, "b", 4096, "size of the buffer for incoming requests")
	flag.IntVar(&opts.WorkerCount, "w", 1, "number of workers to handling requests")

	flag.Parse()

	return opts
}

func receiver(
	listener net.Listener,
	logger log.Logger,
	c chan<- queue.Payload,
	stop <-chan bool,
) {
	for {
		select {
		case <-stop:
			logger.Debug("receiver stopping")

			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				logger.WithError(err).Warn("failed to accept connection")
			}
			defer conn.Close()

			logger.Debug("accepting new connection")

			c <- conn
		}
	}
}

func worker(
	wg *sync.WaitGroup,
	remote net.Conn,
	logger log.Logger,
	bufferSize int,
	c <-chan queue.Payload,
) {
	defer wg.Done()

	for conn := range c {
		logger := logger.WithFields(log.Fields{
			"src": conn.RemoteAddr(),
		})

		for {
			buf := make([]byte, bufferSize)

			length, err := workerRead(logger, conn, buf)
			if err != nil {
				if errors.Is(err, io.EOF) {
					logger.Info("client connection closed")

					break
				}

				logger.WithError(err).Warn("failed reading from client")
			}

			buf = buf[:length]

			for {
				length, err := remote.Write(buf)

				if length > 0 && length < len(buf) {
					buf = buf[length-1:]

					logger.WithFields(log.Fields{
						"len": length,
						"tts": timeToSleep,
					}).Debug("partial write success, sleeping")
					time.Sleep(timeToSleep)

					continue
				}

				if err != nil {
					if errors.Is(err, io.EOF) {
						logger.Info("remote connection closed")

						// No sense continuing to forward data
						return
					}

					logger.WithError(err).Warn("failed writing to remote")

					break
				}

				logger.WithField("len", length).Debug("wrote socket data")
				logger.WithField("len", length).Infof("%v -> %v", conn.LocalAddr(), remote.RemoteAddr())

				break
			}
		}
	}
}

func workerRead(
	logger log.Logger,
	conn queue.Payload,
	buf []byte,
) (int, error) {
	length, err := conn.Read(buf)

	if length == 0 && err == nil {
		logger.WithField("tts", timeToSleep).Debug("empty response from client, sleeping")
		time.Sleep(timeToSleep)

		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	logger.WithField("len", length).Debug("read socket data")
	logger.WithField("len", length).Infof("%v <- %v", conn.LocalAddr(), conn.RemoteAddr())

	return length, nil
}

func main() {
	opts := parseArgs()

	ctx := context.Background()

	var level log.LogLevel

	switch {
	case opts.Debug:
		level = log.Debug
	case opts.Verbose:
		level = log.Info
	case opts.Quiet:
		level = log.Error
	default:
		level = log.Warning
	}

	Log := log.NewLogger(ctx, level)

	listener, err := net.Listen("tcp", opts.LocalAddr)
	if err != nil {
		Log.WithError(err).Fatalf("failed to open listener socket")
	}
	defer listener.Close()

	Log.Debugf("listening on %v", opts.LocalAddr)

	remote, err := net.Dial("tcp", opts.RemoteAddr)
	if err != nil {
		Log.WithError(err).Fatalf("failed to connect to remote addr")
	}
	defer remote.Close()

	Log.Debugf("connected to %v", opts.RemoteAddr)

	wg := sync.WaitGroup{}

	q := queue.NewQueue(opts.WorkerCount, opts.BufferSize, opts.Timeout)

	for n := 0; n < opts.WorkerCount; n++ {
		logger := Log.WithFields(log.Fields{
			"component": "worker",
			"workerId":  n,
		})

		wg.Add(1)

		go worker(&wg, remote, logger, opts.BufferSize, q.Out())
	}

	Log.Debug("starting multiplexer channel")

	q.Open()
	defer q.Close()

	logger := Log.WithFields(log.Fields{
		"component": "receiver",
	})

	stop := make(chan bool)
	go receiver(listener, logger, q.In(), stop)

	wg.Wait()
}
