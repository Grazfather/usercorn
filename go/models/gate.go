package models

import (
	"sync"
)

type waiter struct {
	sync.Mutex
	cb []chan int
}

// Add a waiter channel to the cb list and return it
func (w *waiter) Add() chan int {
	w.Lock()
	ret := make(chan int)
	w.cb = append(w.cb, ret)
	w.Unlock()
	return ret
}

// Truncate removes all waiter channels
func (w *waiter) Truncate() {
	if w.cb != nil {
		w.cb = w.cb[:0]
	}
}

// Notify signals every waiter callback and then truncate the cb list
func (w *waiter) Notify() {
	w.Lock()
	for _, c := range w.cb {
		c <- 1
	}
	w.Truncate()
	w.Unlock()
}

type Gate struct {
	sync.Mutex
	wg sync.WaitGroup

	start, stop waiter
}

func (g *Gate) Start() {
	g.Lock()
	g.start.Notify()
	g.wg.Wait()
}

func (g *Gate) Stop() {
	g.stop.Notify()
	g.Unlock()
	g.wg.Wait()
}

// StopLock will wait for usercorn to stop and then block it from running
func (g *Gate) StopLock() {
	g.wg.Add(1)
	<-g.stop.Add()
	g.Lock()
	g.wg.Done()
}

// UnlockStart will unblock usercorn from running and then wait for it to start
func (g *Gate) UnlockStart() {
	block := g.start.Add()
	g.Unlock()
	<-block
}

// UnlockStop will unblock usercorn from running and then wait for it to stop
func (g *Gate) UnlockStop() {
	block := g.stop.Add()
	g.Unlock()
	<-block
}

// UnlockStopRelock will unblock usercorn from running, wait for it to stop,
// and immediately relock
func (g *Gate) UnlockStopRelock() {
	start := g.stop.Add()
	stop := g.stop.Add()
	g.Unlock()
	<-start
	g.wg.Add(1)
	<-stop
	g.Lock()
	g.wg.Done()
}
