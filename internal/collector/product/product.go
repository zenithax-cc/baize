package product

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type collectTask struct {
	name string
	fn   func() error
}

func New() *Product {
	return &Product{}
}

func (p *Product) Collect(ctx context.Context) error {
	tasks := []collectTask{
		{name: "kernel", fn: p.collectKernel},
		{name: "distribution", fn: p.collectDistribution},
		{name: "bios", fn: p.collectBIOS},
		{name: "baseboard", fn: p.collectBaseBoard},
		{name: "chassis", fn: p.collectChassis},
	}

	return p.runTasksConcurently(ctx, tasks)
}

func (p *Product) runTasksConcurently(ctx context.Context, tasks []collectTask) error {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	wg.Add(len(tasks))

	for _, task := range tasks {
		go func(t collectTask) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", t.name, ctx.Err()))
				mu.Unlock()
				return
			default:
			}
			if err := t.fn(); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", t.name, err))
				mu.Unlock()
			}
		}(task)
	}

	wg.Wait()

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
