package concurrency_test

import (
	"context"
	"goutils/concurrency"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func ptr[T any](v T) *T {
	return &v
}

func TestRunParallelOrdered(t *testing.T) {
	tests := []struct {
		name           string
		timeout        time.Duration
		fns            []func() int
		expectedResult []*int
	}{
		{
			name:    "All functions succeed",
			timeout: 1000 * time.Millisecond,
			fns: []func() int{
				func() int { time.Sleep(10 * time.Millisecond); return 1 },
				func() int { time.Sleep(20 * time.Millisecond); return 2 },
				func() int { time.Sleep(30 * time.Millisecond); return 3 },
			},
			expectedResult: []*int{ptr(1), ptr(2), ptr(3)},
		},
		{
			name:    "Timeout occurs before all functions complete",
			timeout: 150 * time.Millisecond,
			fns: []func() int{
				func() int { time.Sleep(100 * time.Millisecond); return 1 },
				func() int { time.Sleep(200 * time.Millisecond); return 2 },
				func() int { time.Sleep(50 * time.Millisecond); return 3 },
			},
			expectedResult: []*int{ptr(1), nil, ptr(3)},
		},
		{
			name:           "Empty function list",
			timeout:        14 * time.Millisecond,
			fns:            []func() int{},
			expectedResult: []*int{},
		},
		{
			name:    "Zero timeout",
			timeout: 0,
			fns: []func() int{
				func() int { time.Sleep(100 * time.Millisecond); return 1 },
				func() int { time.Sleep(200 * time.Millisecond); return 2 },
				func() int { time.Sleep(50 * time.Millisecond); return 3 },
			},
			expectedResult: []*int{nil, nil, nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()
			results := concurrency.RunParallelOrdered(ctx, tt.fns...)
			assert.Equal(t, tt.expectedResult, results)
		})
	}
}

// Стоит обратить внимание, что тест при низких временах может быть нестабильным
// Так как есть накладные расходы на все остальное кроме времени сами функций,
// и результатов может в канал может больше
func TestRunParallelFiltered(t *testing.T) {
	tests := []struct {
		name           string
		timeout        time.Duration
		fns            []func() int
		expectedResult []struct {
			Val int
			Idx int
		}
	}{
		{
			name:    "All functions succeed",
			timeout: 1000 * time.Millisecond,
			fns: []func() int{
				func() int { time.Sleep(100 * time.Millisecond); return 1 },
				func() int { time.Sleep(200 * time.Millisecond); return 2 },
				func() int { time.Sleep(300 * time.Millisecond); return 3 },
			},
			expectedResult: []struct {
				Val int
				Idx int
			}{{1, 0}, {2, 1}, {3, 2}},
		},
		{
			name:    "Timeout occurs before all functions complete",
			timeout: 150 * time.Millisecond,
			fns: []func() int{
				func() int { time.Sleep(100 * time.Millisecond); return 1 },
				func() int { time.Sleep(200 * time.Millisecond); return 2 },
				func() int { time.Sleep(50 * time.Millisecond); return 3 },
			},
			expectedResult: []struct {
				Val int
				Idx int
			}{{3, 2}, {1, 0}},
		},
		{
			name:    "Empty function list",
			timeout: 14 * time.Millisecond,
			fns:     []func() int{},
			expectedResult: []struct {
				Val int
				Idx int
			}{},
		},
		{
			name:    "Zero timeout",
			timeout: 0,
			fns: []func() int{
				func() int { time.Sleep(10 * time.Millisecond); return 1 },
				func() int { time.Sleep(20 * time.Millisecond); return 2 },
				func() int { time.Sleep(5 * time.Millisecond); return 3 },
			},
			expectedResult: []struct {
				Val int
				Idx int
			}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()
			results := concurrency.RunParallelFiltered(ctx, tt.fns...)
			assert.Equal(t, tt.expectedResult, results)
		})
	}
}

func TestRunParallelMostPriority(t *testing.T) {
	tests := []struct {
		name        string
		fns         []func() int
		priorities  []int
		timeout     time.Duration
		isSuccessFn func(v int) bool
		want        *int
		wantErr     bool
	}{
		{
			name: "successful execution with different priorities",
			fns: []func() int{
				func() int { return 1 },
				func() int { return 2 },
				func() int { return 3 },
			},
			timeout:     100 * time.Millisecond,
			priorities:  []int{1, 2, 3},
			isSuccessFn: func(v int) bool { return true },
			want:        ptr(1),
			wantErr:     false,
		},
		{
			name: "failed execution with different priorities",
			fns: []func() int{
				func() int { return 1 },
				func() int { return 2 },
				func() int { return 3 },
			},
			timeout:     100 * time.Millisecond,
			priorities:  []int{1, 2, 3},
			isSuccessFn: func(v int) bool { return false },
			want:        nil,
			wantErr:     true,
		},
		{
			name: "functions with same priority", // таймауты нужны так как результат без них непредсказуем
			fns: []func() int{
				func() int { return 1 },
				func() int { time.Sleep(1 * time.Millisecond); return 2 },
				func() int { time.Sleep(2 * time.Millisecond); return 3 },
			},
			timeout:     100 * time.Millisecond,
			priorities:  []int{1, 1, 1},
			isSuccessFn: func(v int) bool { return true },
			want:        ptr(1),
			wantErr:     false,
		},
		{
			name:        "cancellation of context",
			fns:         []func() int{func() int { return 1 }},
			priorities:  []int{1},
			isSuccessFn: func(v int) bool { return true },
			want:        nil,
			wantErr:     true,
		},
		{
			name:        "nil priority",
			fns:         []func() int{func() int { return 1 }},
			priorities:  nil,
			isSuccessFn: func(v int) bool { return true },
			want:        nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			got, err := concurrency.RunParallelMostPriority(ctx, tt.isSuccessFn, tt.priorities, tt.fns...)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
