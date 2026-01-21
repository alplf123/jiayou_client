package list

import (
	"reflect"
	"sync"
)

type List[T any] struct {
	arr []T
	lck sync.Mutex
}
type FilterFunc[T any] func(T) bool
type IterFunc[Z int, T any] func(Z, T) bool

func (a *List[T]) Size() int {
	a.lck.Lock()
	defer a.lck.Unlock()
	return a.size()
}

func (a *List[T]) size() int {
	return len(a.arr)
}
func (a *List[T]) Push(any T) *List[T] {
	a.lck.Lock()
	defer a.lck.Unlock()
	return a.push(any)
}
func (a *List[T]) push(any T) *List[T] {
	a.arr = append(a.arr, any)
	return a
}
func (a *List[T]) Pop() T {
	a.lck.Lock()
	defer a.lck.Unlock()
	var ret T
	var size = a.size()
	if size > 0 {
		ret = a.arr[size-1]
		a.removeI(size - 1)
	}
	return ret
}
func (a *List[T]) PopLeft() T {
	a.lck.Lock()
	defer a.lck.Unlock()
	var ret T
	var size = a.size()
	if size > 0 {
		ret = a.arr[0]
		a.removeI(0)
	}
	return ret
}
func (a *List[T]) Index(index int) T {
	a.lck.Lock()
	defer a.lck.Unlock()
	var result T
	if index >= a.size() || index < 0 {
		return result
	}
	result = a.arr[index]
	return result
}
func (a *List[T]) Top() T {
	a.lck.Lock()
	defer a.lck.Unlock()
	var result T
	if a.size() > 0 {
		result = a.arr[0]
	}
	return result
}
func (a *List[T]) Last() T {
	a.lck.Lock()
	defer a.lck.Unlock()
	var result T
	if a.size() > 0 {
		result = a.arr[a.size()-1]
	}
	return result
}
func (a *List[T]) Clear() {
	a.lck.Lock()
	defer a.lck.Unlock()
	a.iter(func(i int, t T) bool {
		a.removeRef(i)
		return true
	})
	a.arr = a.arr[:0]
}
func (a *List[T]) Contain(any T) bool {
	a.lck.Lock()
	defer a.lck.Unlock()
	var index = -1
	a.iter(func(ii int, aa T) bool {
		if reflect.TypeOf(any).Comparable() {
			v1, v2 := reflect.ValueOf(any).Interface(), reflect.ValueOf(aa).Interface()
			if v1 == v2 {
				index = ii
				return true
			}
		}
		return false
	})
	return index != -1
}
func (a *List[T]) ContainF(filter FilterFunc[T]) bool {
	a.lck.Lock()
	defer a.lck.Unlock()
	var index = -1
	a.iter(func(i int, t T) bool {
		if filter(t) {
			index = i
			return true
		}
		return false
	})
	return index != -1

}

func (a *List[T]) Filter(filter FilterFunc[T]) []T {
	a.lck.Lock()
	defer a.lck.Unlock()
	var ret []T
	a.iter(func(index int, any T) bool {
		if filter(any) {
			ret = append(ret, any)
		}
		return false
	})
	return ret
}
func (a *List[T]) Iter(iter IterFunc[int, T]) {
	a.lck.Lock()
	defer a.lck.Unlock()
	a.iter(iter)
}
func (a *List[T]) iter(iter IterFunc[int, T]) {
	for i, v := range a.arr {
		if iter(i, v) {
			return
		}
	}
}

func (a *List[T]) Remove(any T) *List[T] {
	a.lck.Lock()
	defer a.lck.Unlock()
	if !reflect.ValueOf(any).IsValid() {
		return a
	}
	var i int = -1
	a.iter(func(ii int, aa T) bool {
		if reflect.TypeOf(any).Comparable() {
			v1, v2 := reflect.ValueOf(any).Interface(), reflect.ValueOf(aa).Interface()
			if v1 == v2 {
				i = ii
				return true
			}
		}
		return false
	})
	if i != -1 {
		a.removeI(i)
	}
	return a

}
func (a *List[T]) removeI(index int) *List[T] {
	if index >= a.size() || index < 0 {
		return a
	}
	//remove reference
	a.removeRef(index)
	a.arr = append(a.arr[:index], a.arr[index+1:]...)
	return a

}
func (a *List[T]) removeRef(index int) {
	if index >= a.size() || index < 0 {
		return
	}
	val := reflect.ValueOf(a.arr).Index(index)
	if val.Kind() == reflect.Pointer {
		val.Set(reflect.Zero(val.Type()))
	}
}
func (a *List[T]) RemoveI(index int) *List[T] {
	a.lck.Lock()
	defer a.lck.Unlock()
	return a.removeI(index)
}
func (a *List[T]) RemoveF(filter FilterFunc[T]) *List[T] {
	a.lck.Lock()
	defer a.lck.Unlock()
	var i []int
	a.iter(func(index int, any T) bool {
		if filter(any) {
			i = append(i, index)
			return true
		}
		return false
	})
	if len(i) == 0 {
		return a
	}
	var c int
	for _, v := range i {
		a.removeI(v - c)
		c++
	}
	return a

}
func (a *List[T]) PushRange(rge ...T) *List[T] {
	a.lck.Lock()
	defer a.lck.Unlock()
	for _, t := range rge {
		a.push(t)
	}
	return a

}
func (a *List[T]) PushArray(arr *List[T]) *List[T] {
	a.lck.Lock()
	defer a.lck.Unlock()
	if a.size() > 0 {
		a.arr = append(a.arr, arr.arr...)
	}
	return a

}
func (a *List[T]) ToArray() []T {
	a.lck.Lock()
	defer a.lck.Unlock()
	var arr []T
	if a.size() > 0 {
		arr = make([]T, a.size())
		copy(arr, a.arr)
	}
	return arr

}

func New[T any]() *List[T] {
	return &List[T]{}
}
