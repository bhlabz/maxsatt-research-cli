package utils

import "sync"

var mu sync.Mutex

func ExecuteWithMutex(fn func()) {
	mu.Lock()
	defer mu.Unlock()
	fn()
}

var mu2 sync.Mutex

func ExecuteWithMutex2(fn func()) {
	mu2.Lock()
	defer mu2.Unlock()
	fn()
}
