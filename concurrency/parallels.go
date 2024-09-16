package concurrency

import (
	"context"
	"fmt"
	"sort"
)

type taskStatus int

const (
	taskStatusWaiting taskStatus = 0 // важно, чтобы значением было 0 т.к. 0 дефолтное значение инта
	taskStatusSuccess taskStatus = 1
	taskStatusFailed  taskStatus = 2
)

// RunParallelMostPriority зпаралельно запускает функции
// Возвращает результат самой приоритетной функции среди успешных
// приоритет берется из массива priorities (чем ниже число, тем выше приоритет)
// например для приоритетов [1, 8, 10] вернется результат функции с приоритетом 1 (при улсовии, что она выполнилась успешно)
// если приоритет будет одинаковый, то вернется результат самой быстрой их них
func RunParallelMostPriority[T any](ctx context.Context, isSuccesFn func(v T) bool, priorities []int, fns ...func() T) (*T, error) {
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// преобразует приоритеты в ранги и если надо дозаполняет
	makePriorities := func(priorities []int, fnsCount int) []int {
		res := make([]int, fnsCount)
		priorities = arr2RankArray(priorities)

		length := len(priorities)
		if length > fnsCount {
			length = fnsCount
		}

		lowestPriproty := -1
		for i := 0; i < length; i++ {
			p := priorities[i]

			if lowestPriproty < p {
				lowestPriproty = p
			}

			if i < len(priorities) {
				res[i] = p
			}
		}

		// все остальные функции одинаково низкий приоритет будут иметь
		for i := len(priorities); i < fnsCount; i++ {
			res[i] = lowestPriproty + 1
		}

		return res
	}

	// функция, чтобы найти первый успешный и самый приоритетный результат
	findResolvedTop := func(statuses []taskStatus, results []*T) (*T, bool) {
		length := len(statuses) // должно совпадать с длиной results

		for i := 0; i < length; i++ {
			res := results[i]
			status := statuses[i]

			// пока предыдущая задача не выполнена, еще неизвестно можно ли возвращать текущую
			if status == taskStatusWaiting {
				return nil, false
			}

			if status == taskStatusSuccess && res != nil {
				return res, true
			}

			if status == taskStatusFailed {
				continue
			}
		}

		return nil, false
	}

	priorities = makePriorities(priorities, len(fns))

	statuses := make([]taskStatus, len(fns))
	results := make([]*T, len(fns))

	resChan := Funcs2Channels(cancelCtx, fns...)
	for v := range resChan {
		idx := priorities[v.Idx]

		isSucces := isSuccesFn(v.Val)
		if isSucces {
			statuses[idx] = taskStatusSuccess
		} else {
			statuses[idx] = taskStatusFailed
		}
		results[idx] = &v.Val

		res, ok := findResolvedTop(statuses, results)
		if ok {
			return res, nil
		}

	}

	for i := 0; i < len(fns); i++ {
		if statuses[i] == taskStatusSuccess {
			return results[i], nil
		}
	}

	return nil, fmt.Errorf("all tasks not success resolve")
}

// arr2RankArray превращает массив в ранговый
// пример: [5, 10, 3] => [1, 2, 0]
// самый низкий номер стартует с 0
func arr2RankArray(arr []int) []int {
	sortedArr := make([]int, len(arr))
	copy(sortedArr, arr)

	sort.Ints(sortedArr)

	rankMap := make(map[int]int)
	for i, val := range sortedArr {
		_, ok := rankMap[val]
		if ok {
			continue
		}

		rankMap[val] = i
	}

	rankedArr := make([]int, len(arr))
	for i, val := range arr {
		rankedArr[i] = rankMap[val]
	}

	return rankedArr
}

// RunParallelOrdered выполняет функции параллельно, пока не отменят контекст
// Порядок результатов совпадает с порядком функций
// (если значение nil, значит контекст отменился и функция не успела выполниться)
// Важно, чтобы принимаемые функции не кидали панику!!!
// В сумме на все функции создается len(fns) + 1 горутина
func RunParallelOrdered[T any](ctx context.Context, fns ...func() T) []*T {
	resultChan := Funcs2Channels(ctx, fns...)

	results := make([]*T, len(fns))
	for item := range resultChan {
		results[item.Idx] = &item.Val
	}

	return results
}

// RunParallelFiltered как RunParallelOrdered,
// только возвращает массив всех успешных результатов + индекс исходной функции
func RunParallelFiltered[T any](ctx context.Context, fns ...func() T) []struct {
	Val T
	Idx int
} {
	resultChan := Funcs2Channels(ctx, fns...)

	results := make([]struct {
		Val T
		Idx int
	}, 0)
	for item := range resultChan {
		v := struct {
			Val T
			Idx int
		}{
			Val: item.Val,
			Idx: item.Idx,
		}
		results = append(results, v)
	}

	return results
}
