package concurrency

import (
	"context"
)

// Wrap2ChanFn обернуть в функциюб которую можно отменить по контексту
func Wrap2ChanFn[T any](ctx context.Context, f func() T) func() <-chan T {
	ch := make(chan T)

	return func() <-chan T {
		go func() {
			defer close(ch)

			select {
			case <-ctx.Done():
				return
			case ch <- f():
				return
			}
		}()

		return ch
	}
}

// MergeChannels cливает каналы в 1, при этом канал возвращает значение с индексом исходного канала
func MergeChannels[T any](ctx context.Context, channels ...<-chan T) <-chan struct {
	Val T
	Idx int
} {
	out := make(chan struct {
		Val T
		Idx int
	})

	go func() {
		defer close(out)

		openChannelsCount := len(channels)
		openChannels := make([]bool, len(channels))
		for i := range openChannels {
			openChannels[i] = true
		}

		for openChannelsCount > 0 {
			for i, ch := range channels {
				if !openChannels[i] {
					continue
				}

				select {
				case <-ctx.Done():
					return
				case val, ok := <-ch:
					if !ok {
						openChannels[i] = false
						openChannelsCount--
						continue
					}
					out <- struct {
						Val T
						Idx int
					}{Val: val, Idx: i}
				default:
				}
			}
		}
	}()
	return out
}

// Funcs2Channels запускает функции в горутинах и возвращает канал с результатами всех функций
// в результате так же будет индекс исходной функции
// создает кол-во горутин len(funcs) + 1
func Funcs2Channels[T any](ctx context.Context, funcs ...func() T) <-chan struct {
	Val T
	Idx int
} {
	fChans := make([]<-chan T, 0)
	for _, fn := range funcs {
		f := Wrap2ChanFn(ctx, fn)
		fChans = append(fChans, f())
	}

	return MergeChannels(ctx, fChans...)
}
