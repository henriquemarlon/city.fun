package workerpool

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type TestJob struct {
	ID   int
	Data string
}

type TestResult struct {
	JobID     int
	Processed string
	Success   bool
}

func TestWorkerPool_Start(t *testing.T) {
	processFunc := func(ctx context.Context, job Job) Result {
		testJob := job.(TestJob)

		processed := "Processed: " + testJob.Data

		return TestResult{
			JobID:     testJob.ID,
			Processed: processed,
			Success:   true,
		}
	}

	config := Config{
		WorkerCount: 2,
	}

	pool := New(processFunc, config)

	assert.False(t, pool.IsRunning())

	inputCh := make(chan Job)

	ctx := context.Background()
	resultCh, err := pool.Start(ctx, inputCh)
	assert.NoError(t, err)
	assert.NotNil(t, resultCh)

	assert.True(t, pool.IsRunning())

	job := TestJob{
		ID:   1,
		Data: "Test Job",
	}
	inputCh <- job

	result := <-resultCh
	testResult := result.(TestResult)
	assert.Equal(t, 1, testResult.JobID)
	assert.Equal(t, "Processed: Test Job", testResult.Processed)
	assert.True(t, testResult.Success)

	err = pool.Stop()
	assert.NoError(t, err)

	assert.False(t, pool.IsRunning())
}

func TestWorkerPool_Start_AlreadyRunning(t *testing.T) {
	processFunc := func(ctx context.Context, job Job) Result {
		return nil
	}

	config := Config{
		WorkerCount: 1,
	}

	pool := New(processFunc, config)

	inputCh := make(chan Job)

	ctx := context.Background()
	resultCh, err := pool.Start(ctx, inputCh)
	assert.NoError(t, err)
	assert.NotNil(t, resultCh)

	resultCh2, err := pool.Start(ctx, inputCh)
	assert.Error(t, err)
	assert.Equal(t, ErrWorkerPoolAlreadyRunning, err)
	assert.Nil(t, resultCh2)

	err = pool.Stop()
	assert.NoError(t, err)
}

func TestWorkerPool_Stop_NotRunning(t *testing.T) {
	processFunc := func(ctx context.Context, job Job) Result {
		return nil
	}

	config := Config{
		WorkerCount: 1,
	}

	pool := New(processFunc, config)

	err := pool.Stop()
	assert.Error(t, err)
	assert.Equal(t, ErrWorkerPoolNotRunning, err)
}

func TestWorkerPool_MultipleJobs(t *testing.T) {
	processFunc := func(ctx context.Context, job Job) Result {
		testJob := job.(TestJob)

		processed := "Processed: " + testJob.Data

		return TestResult{
			JobID:     testJob.ID,
			Processed: processed,
			Success:   true,
		}
	}

	config := Config{
		WorkerCount: 3,
	}

	pool := New(processFunc, config)

	inputCh := make(chan Job)

	ctx := context.Background()
	resultCh, err := pool.Start(ctx, inputCh)
	assert.NoError(t, err)

	numJobs := 10

	go func() {
		for i := 0; i < numJobs; i++ {
			job := TestJob{
				ID:   i,
				Data: "Job " + string(rune('A'+i)),
			}
			inputCh <- job
		}
		close(inputCh)
	}()

	var results []TestResult
	var resultsMutex sync.Mutex

	for result := range resultCh {
		testResult := result.(TestResult)
		resultsMutex.Lock()
		results = append(results, testResult)
		resultsMutex.Unlock()
	}

	assert.Equal(t, numJobs, len(results))

	assert.False(t, pool.IsRunning())
}

func TestWorkerPool_ContextCancellation(t *testing.T) {
	processFunc := func(ctx context.Context, job Job) Result {
		testJob := job.(TestJob)

		select {
		case <-time.After(100 * time.Millisecond):
			processed := "Processed: " + testJob.Data

			return TestResult{
				JobID:     testJob.ID,
				Processed: processed,
				Success:   true,
			}
		case <-ctx.Done():
			return TestResult{
				JobID:     testJob.ID,
				Processed: "Cancelled",
				Success:   false,
			}
		}
	}

	config := Config{
		WorkerCount: 2,
	}

	pool := New(processFunc, config)

	inputCh := make(chan Job)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultCh, err := pool.Start(ctx, inputCh)
	assert.NoError(t, err)

	for i := 0; i < 2; i++ {
		job := TestJob{
			ID:   i,
			Data: "Job " + string(rune('A'+i)),
		}
		inputCh <- job
	}

	time.Sleep(50 * time.Millisecond)
	cancel()

	close(inputCh)

	err = pool.Stop()

	var results []TestResult
	timeout := time.After(200 * time.Millisecond)

collectResults:
	for {
		select {
		case result, ok := <-resultCh:
			if !ok {
				break collectResults
			}
			testResult := result.(TestResult)
			results = append(results, testResult)
		case <-timeout:
			t.Log("Timeout receiving results")
			break collectResults
		}
	}

	assert.False(t, pool.IsRunning())
}
