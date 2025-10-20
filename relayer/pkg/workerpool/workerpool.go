package workerpool

import (
	"context"
	"log/slog"
	"os"
	"sync"
)

type Job interface{}

type Result interface{}

type ProcessFunc func(ctx context.Context, job Job) Result

type WorkerPool interface {
	Start(ctx context.Context, inputCh <-chan Job) (<-chan Result, error)

	Stop() error

	IsRunning() bool
}

type State int

const (
	StateIdle State = iota
	StateRunning
	StateStopping
)

func (s State) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateRunning:
		return "running"
	case StateStopping:
		return "stopping"
	default:
		return "unknown"
	}
}

type Config struct {
	WorkerCount int
	Logger *slog.Logger
}

func DefaultConfig() Config {
	return Config{
		WorkerCount: 1,
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
	}
}

type workerPool struct {
	processFunc ProcessFunc
	workerCount int
	stopCh      chan struct{}
	stopWg      sync.WaitGroup
	state       State
	stateMutex  sync.Mutex
	logger      *slog.Logger
}

func New(processFunc ProcessFunc, config Config) WorkerPool {
	if config.WorkerCount <= 0 {
		config.WorkerCount = 1
	}

	if config.Logger == nil {
		config.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	return &workerPool{
		processFunc: processFunc,
		workerCount: config.WorkerCount,
		stopCh:      make(chan struct{}),
		state:       StateIdle,
		logger:      config.Logger,
	}
}

func (wp *workerPool) Start(ctx context.Context, inputCh <-chan Job) (<-chan Result, error) {
	wp.stateMutex.Lock()
	defer wp.stateMutex.Unlock()

	if wp.state != StateIdle {
		return nil, ErrWorkerPoolAlreadyRunning
	}

	resultCh := make(chan Result)
	wp.state = StateRunning
	wp.stopCh = make(chan struct{})

	wp.stopWg.Add(wp.workerCount)
	for i := 0; i < wp.workerCount; i++ {
		go wp.worker(ctx, i, inputCh, resultCh)
	}

	go func() {
		wp.stopWg.Wait()
		close(resultCh)
		wp.stateMutex.Lock()
		wp.state = StateIdle
		wp.stateMutex.Unlock()
	}()

	return resultCh, nil
}

func (wp *workerPool) Stop() error {
	wp.stateMutex.Lock()
	defer wp.stateMutex.Unlock()

	if wp.state != StateRunning {
		return ErrWorkerPoolNotRunning
	}

	wp.state = StateStopping
	close(wp.stopCh)
	wp.stopWg.Wait()
	wp.state = StateIdle
	return nil
}

func (wp *workerPool) IsRunning() bool {
	wp.stateMutex.Lock()
	defer wp.stateMutex.Unlock()
	return wp.state == StateRunning
}

func (wp *workerPool) worker(ctx context.Context, id int, inputCh <-chan Job, resultCh chan<- Result) {
	defer wp.stopWg.Done()

	wp.logger.Info("Starting worker", "worker_id", id)

	for {
		select {
		case <-wp.stopCh:
			wp.logger.Info("Worker stopped", "worker_id", id)
			return
		case <-ctx.Done():
			wp.logger.Info("Context cancelled, stopping worker", "worker_id", id)
			return
		case job, ok := <-inputCh:
			if !ok {
				wp.logger.Info("Input channel closed, stopping worker", "worker_id", id)
				return
			}

			result := wp.processFunc(ctx, job)
			select {
			case resultCh <- result:
			case <-wp.stopCh:
				wp.logger.Info("Worker stopped during result sending", "worker_id", id)
				return
			case <-ctx.Done():
				wp.logger.Info("Context cancelled during result sending", "worker_id", id)
				return
			}
		}
	}
}