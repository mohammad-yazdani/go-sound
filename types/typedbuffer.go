// A circular buffer data type for generic values.
package types

import (
	"sync"
)

// Buffer holds the values within the buffer plus a collection of metadata.
type TypedBuffer struct {
	values   []interface{}
	capacity int
	size     int
	at       int
	lock     sync.Mutex
	finished bool
}

// NewTypedBuffer creates a new circular buffer of a given maximum size.
func NewTypedBuffer(capacity int) *TypedBuffer {
	b := TypedBuffer{
		make([]interface{}, capacity),
		capacity,
		0, /* size */
		0, /* at */
		sync.Mutex{},
		false, /* finished */
	}
	return &b
}

// Push adds a new value at the end of the buffer.
func (b *TypedBuffer) Push(value interface{}) interface{} {
	b.lock.Lock()

	result := b.values[b.at]
	b.values[b.at] = value

	if b.size < b.capacity {
		b.size++
		result = 0.0
	}

	if b.at+1 < b.capacity {
		b.at = b.at + 1
	} else {
		b.at = 0
	}

	b.lock.Unlock()
	return result
}

// GoPushChannel constantly pushes values from a channel, in a separate thread,
// optionally only sampling 1 every sampleRate values.
func (b *TypedBuffer) GoPushChannel(values <-chan interface{}, sampleRate int) {
	var val interface{}
	ok := true
	b.finished = false
	go func() {
		for {
			if val, ok = <-values; !ok {
				break
			}
			b.Push(val)
			for i := 1; i < sampleRate; i++ {
				if _, ok = <-values; !ok {
					break
				}
			}
		}
		b.finished = true
	}()
}

// GetFromEnd returns the most recent buffer values.
// 0 returns the most recently pushed, the least recent being b.size - 1
func (b *TypedBuffer) GetFromEnd(index int) interface{} {
	b.lock.Lock()
	defer b.lock.Unlock()
	if index < 0 || index >= b.capacity {
		panic("GetFromEnd index out of range")
	} else if index >= b.size {
		// Within range, just not filled yet, to default to zero.
		return 0.0
	}

	index = b.at - index
	if index < 0 {
		index = index + b.capacity
	}
	result := b.values[index]
	return result
}

// IsFull returns whether the buffer is full,
// in that adding more entries will delete older ones.
func (b *TypedBuffer) IsFull() bool {
	return b.size == b.capacity
}

// IsFinished returns whether there is nothing more to be added to the buffer
func (b *TypedBuffer) IsFinished() bool {
	return b.finished
}

// Clear resets the buffer to being empty
func (b *TypedBuffer) Clear() {
	// Simply clamp the size back to zero, don't worry about the existing values.
	b.lock.Lock()
	b.size = 0
	b.lock.Unlock()
}

// Each applies a given function to all the values in the buffer,
// from least recent first, ending at the most recent.
func (b *TypedBuffer) Each(cb func(int, interface{})) {
	b.lock.Lock()
	i := 0
	if !b.IsFull() {
		for i = 0; i < b.size; i++ {
			cb(i, b.values[i])
		}
	} else {
		index := 0
		for i = b.at; i < b.capacity; i++ {
			cb(index, b.values[i])
			index++
		}
		for i = 0; i < b.at; i++ {
			cb(index, b.values[i])
			index++
		}
	}
	b.lock.Unlock()
}
