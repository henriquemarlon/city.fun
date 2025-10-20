package workerpool

import "errors"

var (
	ErrWorkerPoolAlreadyRunning = errors.New("worker pool is already running")

	ErrWorkerPoolNotRunning = errors.New("worker pool is not running")
)