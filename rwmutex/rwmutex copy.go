// //go:build !solution

package rwmutex

// // A RWMutex is a reader/writer mutual exclusion lock.
// // The lock can be held by an arbitrary number of readers or a single writer.
// // The zero value for a RWMutex is an unlocked mutex.
// //
// // If a goroutine holds a RWMutex for reading and another goroutine might
// // call Lock, no goroutine should expect to be able to acquire a read lock
// // until the initial read lock is released. In particular, this prohibits
// // recursive read locking. This is to ensure that the lock eventually becomes
// // available; a blocked Lock call excludes new readers from acquiring the
// // lock.
// type RWMutex struct {
// 	addMutex chan struct{}
// 	minusMutex chan struct{}
// 	notifyMutex chan struct{}
// 	isNotify bool
// 	readerCount int 
// }

// // New creates *RWMutex.
// func New() *RWMutex {
// 	rwMutex := RWMutex{
// 		addMutex : make(chan struct{},1),
// 		minusMutex : make(chan struct{},1),
// 		notifyMutex : make(chan struct{},1),
// 		isNotify:  false,
// 		readerCount: 0,
// 	}

// 	return &rwMutex
// }

// // RLock locks rw for reading.
// //
// // It should not be used for recursive read locking; a blocked Lock
// // call excludes new readers from acquiring the lock. See the
// // documentation on the RWMutex type.
// func (rw *RWMutex) RLock() {
// 	rw.addMutex <- struct{}{}
// 	<- rw.addMutex

//     rw.minusMutex <- struct{}{}
// 	rw.readerCount++
// 	if rw.readerCount > 0 && !rw.isNotify {
// 		rw.notifyMutex <- struct{}{}
// 		rw.isNotify = true
// 	}
// 	<- rw.minusMutex
// }


// // rw.addMutex <- struct{}{}
// // rw.minusMutex <- struct{}{}
// // <- rw.minusMutex
// // <- rw.addMutex

// // RUnlock undoes a single RLock call;
// // it does not affect other simultaneous readers.
// // It is a run-time error if rw is not locked for reading
// // on entry to RUnlock.
// func (rw *RWMutex) RUnlock() {
// 	rw.minusMutex <- struct{}{}
// 	rw.readerCount--

// 	if rw.readerCount == 0 {
// 		<- rw.notifyMutex 
// 		rw.isNotify = false 
// 	}
// 	<- rw.minusMutex

// }

// // Lock locks rw for writing.
// // If the lock is already locked for reading or writing,
// // Lock blocks until the lock is available.
// func (rw *RWMutex) Lock() {
// 	rw.addMutex <- struct{}{}
// 	rw.notifyMutex <- struct{}{}
// }

// // Unlock unlocks rw for writing. It is a run-time error if rw is
// // not locked for writing on entry to Unlock.
// //
// // As with Mutexes, a locked RWMutex is not associated with a particular
// // goroutine. One goroutine may RLock (Lock) a RWMutex and then
// // arrange for another goroutine to RUnlock (Unlock) it.
// func (rw *RWMutex) Unlock() {
// 	<- rw.notifyMutex 
// 	<- rw.addMutex
// }
