package progress

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
)

// RunBatch starts sources concurrently and calls emit for every Update.
func RunBatch(ctx context.Context, sources []Source, emit func(Update)) error {
	return runBatch(ctx, sources, emit, 0, false)
}

func runBatch(ctx context.Context, sources []Source, emit func(Update), limit int, lenient bool) error {
	if emit == nil {
		emit = func(Update) {}
	}
	g, ctx := errgroup.WithContext(ctx)
	updates := make(chan Update, 32)

	g.Go(func() error {
		defer close(updates)
		inner, ctx := errgroup.WithContext(ctx)
		if limit > 0 {
			inner.SetLimit(limit)
		}
		for _, src := range sources {
			src := src
			inner.Go(func() error {
				err := src.Run(ctx, updates)
				if lenient {
					return nil
				}
				return err
			})
		}
		return inner.Wait()
	})

	var emitErr error
	var once sync.Once
	g.Go(func() error {
		for u := range updates {
			emit(u)
			if !lenient && u.Err != nil {
				once.Do(func() { emitErr = u.Err })
			}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}
	if lenient {
		return nil
	}
	return emitErr
}
