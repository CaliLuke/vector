package devtools

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	queueCapacity = 2000
	batchCapacity = 100
)

const logEndpoint = "http://127.0.0.1:54318/v1/logs"

var dockerServiceSuffix = regexp.MustCompile(`-\d+$`)

type child struct {
	cmd    *exec.Cmd
	docker bool
	done   chan struct{}
	err    error
}

// RunLocalDevelopment starts frontend tasks and tails local containers that do
// not export their own logs to Logal.
// Interrupts and termination signals are forwarded to each process group.
func RunLocalDevelopment(ctx context.Context, root string, stdout, stderr io.Writer) int {
	return runLocalDevelopment(ctx, root, stdout, stderr, logEndpoint)
}

func runLocalDevelopment(ctx context.Context, root string, stdout, stderr io.Writer, endpoint string) int {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()
	queue := make(chan map[string]any, queueCapacity)
	var readers sync.WaitGroup
	readerCtx, stopReaders := context.WithCancel(context.Background())
	defer stopReaders()
	var warnMu sync.Mutex
	lastQueueWarning := time.Time{}
	addRecord := func(record map[string]any) {
		if record == nil {
			return
		}
		select {
		case queue <- record:
		default:
			select {
			case <-queue:
			default:
			}
			select {
			case queue <- record:
			default:
			}
			warnMu.Lock()
			if time.Since(lastQueueWarning) > 30*time.Second {
				fmt.Fprintln(stderr, "Logal queue full; dropping oldest local log.")
				lastQueueWarning = time.Now()
			}
			warnMu.Unlock()
		}
	}

	writerDone := make(chan struct{})
	go exportLoop(readerCtx, queue, endpoint, stderr, writerDone)
	start := func(docker bool) (*child, error) {
		args := []string{"run", "dev:frontend"}
		name := "pnpm"
		if docker {
			name = "docker"
			args = []string{"compose", "-p", "plane", "-f", "docker-compose-local.yml", "logs", "--follow", "--since", "1m", "--no-color", "--timestamps", "go-migrator", "proxy"}
		}
		cmd := exec.Command(name, args...)
		cmd.Dir = root
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		pipe, sink, err := os.Pipe()
		if err != nil {
			return nil, err
		}
		cmd.Stdout = sink
		cmd.Stderr = sink
		if err := cmd.Start(); err != nil {
			pipe.Close()
			sink.Close()
			return nil, err
		}
		_ = sink.Close()
		c := &child{cmd: cmd, docker: docker, done: make(chan struct{})}
		readers.Add(1)
		go func() {
			defer readers.Done()
			scanChild(readerCtx, pipe, docker, stdout, addRecord)
		}()
		go func() {
			c.err = cmd.Wait()
			close(c.done)
		}()
		return c, nil
	}

	dockerChild, err := start(true)
	if err != nil {
		fmt.Fprintf(stderr, "Start Docker log capture: %v\n", err)
		stopReaders()
		close(queue)
		<-writerDone
		return 1
	}
	frontendChild, err := start(false)
	if err != nil {
		fmt.Fprintf(stderr, "Start frontend tasks: %v\n", err)
		terminateChildren([]*child{dockerChild}, 10*time.Second)
		stopReaders()
		readers.Wait()
		close(queue)
		<-writerDone
		return 1
	}
	var exit int
	var dockerRetry <-chan time.Time
	for {
		var dockerDone <-chan struct{}
		if dockerChild != nil {
			dockerDone = dockerChild.done
		}
		select {
		case <-ctx.Done():
			exit = 0
			goto shutdown
		case <-frontendChild.done:
			exit = commandExitCode(frontendChild.err)
			goto shutdown
		case <-dockerDone:
			if ctx.Err() != nil {
				goto shutdown
			}
			fmt.Fprintln(stderr, "Reconnecting local container log capture.")
			dockerChild = nil
			dockerRetry = time.After(2 * time.Second)
		case <-dockerRetry:
			dockerRetry = nil
			var startErr error
			dockerChild, startErr = start(true)
			if startErr != nil {
				fmt.Fprintf(stderr, "Restart Docker log capture: %v\n", startErr)
				dockerRetry = time.After(2 * time.Second)
			}
		}
	}

shutdown:
	terminateChildren([]*child{dockerChild, frontendChild}, 10*time.Second)
	stopReaders()
	readers.Wait()
	close(queue)
	select {
	case <-writerDone:
	case <-time.After(5 * time.Second):
	}
	return exit
}

func commandExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if code := exitErr.ExitCode(); code >= 0 {
			return code
		}
	}
	return 1
}

func terminateChildren(children []*child, timeout time.Duration) {
	for _, c := range children {
		if c == nil || c.cmd.Process == nil {
			continue
		}
		_ = syscall.Kill(-c.cmd.Process.Pid, syscall.SIGTERM)
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for _, c := range children {
		if c == nil {
			continue
		}
		select {
		case <-c.done:
		case <-deadline.C:
			for _, remaining := range children {
				if remaining != nil && remaining.cmd.Process != nil {
					_ = syscall.Kill(-remaining.cmd.Process.Pid, syscall.SIGKILL)
				}
			}
			return
		}
	}
}

func scanChild(ctx context.Context, input io.Reader, docker bool, stdout io.Writer, add func(map[string]any)) {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return
		}
		line := scanner.Text()
		service := "plane-frontend"
		var eventTime time.Time
		if docker {
			prefix, content, ok := strings.Cut(line, "|")
			if !ok {
				continue
			}
			service = "plane-" + dockerServiceSuffix.ReplaceAllString(strings.TrimSpace(prefix), "")
			stamp, _, found := strings.Cut(strings.TrimSpace(content), " ")
			if found {
				if parsed, err := time.Parse(time.RFC3339Nano, stamp); err == nil {
					eventTime = parsed
				} else {
					line = content
				}
			}
		} else {
			fmt.Fprintln(stdout, line)
		}
		add(RecordLog(line, service, eventTime))
	}
}

func exportLoop(ctx context.Context, queue chan map[string]any, endpoint string, stderr io.Writer, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	lastWarning := time.Time{}
	batch := make([]map[string]any, 0, batchCapacity)
	flush := func() bool {
		if len(batch) == 0 {
			return true
		}
		exportCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := ExportLogs(exportCtx, endpoint, batch)
		cancel()
		if err != nil {
			if time.Since(lastWarning) > 30*time.Second {
				fmt.Fprintf(stderr, "Logal export failed: %v\n", err)
				lastWarning = time.Now()
			}
			if ctx.Err() == nil {
				// Retry before newer queued records, dropping oldest queued entries if full.
				for i := len(batch) - 1; i >= 0; i-- {
					select {
					case queue <- batch[i]:
					default:
						select {
						case <-queue:
						default:
						}
						select {
						case queue <- batch[i]:
						default:
						}
					}
				}
			}
			batch = batch[:0]
			return false
		}
		batch = batch[:0]
		return true
	}
	for {
		select {
		case item, ok := <-queue:
			if !ok {
				_ = flush()
				return
			}
			batch = append(batch, item)
			if len(batch) >= batchCapacity {
				if !flush() {
					select {
					case <-time.After(time.Second):
					case <-ctx.Done():
					}
				}
			}
		case <-ticker.C:
			if !flush() {
				select {
				case <-time.After(time.Second):
				case <-ctx.Done():
				}
			}
		case <-ctx.Done():
			// Continue draining until the caller closes the queue.
		}
	}
}
