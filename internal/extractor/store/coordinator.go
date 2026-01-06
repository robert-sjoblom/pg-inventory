package store

import (
	"context"
	"sync"
)

// ConnectionCoordinator ensures max 1 connection per database at a time
type ConnectionCoordinator struct {
	dbSemaphores map[string]chan struct{}
	mu           sync.Mutex
}

func NewConnectionCoordinator() *ConnectionCoordinator {
	return &ConnectionCoordinator{
		dbSemaphores: make(map[string]chan struct{}),
	}
}

/*
	This is perhaps obvious to someone that speaks go fluently, but I'm not that
	person.

	The `mu` mutex protects our map.
	The map is a map of dbname and channels

	Each channel has a buffer of 1

	When Acquire runs, it first acquires the mutex lock
	then it checks if we already have a channel -- if not, we create it.
	It then gets a reference to the channel (sem) and releases the lock.

	Then, as the last step, Acquire sends an empty struct on the channel it
	acquired.
*/

// Acquire blocks until we can connect to the database
func (c *ConnectionCoordinator) Acquire(ctx context.Context, dbName string) error {
	c.mu.Lock()
	if c.dbSemaphores[dbName] == nil {
		// first time we're seeing this database
		c.dbSemaphores[dbName] = make(chan struct{}, 1)
	}
	sem := c.dbSemaphores[dbName]
	c.mu.Unlock()

	select {
	// This will block if buffer is full
	case sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

/*
	This is easier to understand.

	Grab lock. Get semaphore. Listen to semaphore which in turns clears the
	channel's buffer.
*/

// Release frees the connection permit for the database
func (c *ConnectionCoordinator) Release(dbName string) {
	c.mu.Lock()
	sem := c.dbSemaphores[dbName]
	c.mu.Unlock()

	if sem != nil {
		<-sem
	}
}
