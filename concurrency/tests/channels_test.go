package concurrency_test

import (
	"context"
	"goutils/concurrency"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
)

func TestWrap2ChanFn(t *testing.T) {
	tests := []struct {
		name          string
		ctxTimeout    time.Duration
		f             func() int
		expectedValue int
		expectTimeout bool
	}{
		{
			name:          "normal execution",
			ctxTimeout:    100 * time.Millisecond,
			f:             func() int { return 42 },
			expectedValue: 42,
			expectTimeout: false,
		},
		{
			name:          "context cancellation",
			ctxTimeout:    10 * time.Millisecond,
			f:             func() int { time.Sleep(50 * time.Millisecond); return 42 },
			expectedValue: 0,
			expectTimeout: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
			defer cancel()

			resultFunc := concurrency.Wrap2ChanFn(tt.f)
			resultChan := resultFunc()

			select {
			case result := <-resultChan:
				if tt.expectTimeout {
					t.Errorf("expected timeout, but got result: %d", result)
				} else {
					assert.Equal(t, tt.expectedValue, result)
				}
			case <-ctx.Done():
				if !tt.expectTimeout {
					t.Error("expected result, but got timeout")
				}
			}
		})
	}
}

func TestMergeChannels(t *testing.T) {
	type chValues struct {
		val   int
		sleep time.Duration
	}

	tests := []struct {
		name           string
		ctxTimeout     time.Duration
		chValues       [][]chValues
		expectedValues map[int]bool
		expectTimeout  bool
	}{
		{
			name:       "merge channels",
			ctxTimeout: 100 * time.Millisecond,
			chValues: [][]chValues{
				{{1, 10 * time.Millisecond}, {8, 20 * time.Millisecond}},
				{{2, 5 * time.Millisecond}},
			},
			expectedValues: map[int]bool{1: true, 2: true, 8: true},
			expectTimeout:  false,
		},
		{
			name:       "context cancellation",
			ctxTimeout: 50 * time.Millisecond,
			chValues: [][]chValues{
				{{1, 10 * time.Millisecond}, {3, 200 * time.Millisecond}},
				{{2, 5 * time.Millisecond}},
				{{10, 5 * time.Millisecond}},
			},
			expectedValues: map[int]bool{2: true, 1: true, 10: true},
			expectTimeout:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
			defer cancel()

			// просто заполняем каналы
			channels := make([]<-chan int, len(tt.chValues))
			for i, chVals := range tt.chValues {
				ch := make(chan int, len(chVals))
				channels[i] = ch

				go func(ch chan int, chVals []chValues) {
					for _, v := range chVals {
						time.Sleep(v.sleep)
						ch <- v.val
					}
					close(ch)
				}(ch, chVals)
			}

			merged := concurrency.MergeChannels(ctx, channels...)
			results := []int{}
			for item := range merged {
				results = append(results, item.Val)
			}

			resultMap := make(map[int]bool)
			for _, val := range results {
				resultMap[val] = true
			}
			assert.Equal(t, tt.expectedValues, resultMap)
		})
	}
}
