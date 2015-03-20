// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tabletserver

import (
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	log "github.com/golang/glog"
	"github.com/youtube/vitess/go/acl"
	"github.com/youtube/vitess/go/memcache"
	"github.com/youtube/vitess/go/pools"
	"github.com/youtube/vitess/go/stats"
	"github.com/youtube/vitess/go/sync2"
	"golang.org/x/net/context"
)

const statsURL = "/debug/memcache/"

// CreateCacheFunc defines the function signature to create a memcache connection.
type CreateCacheFunc func() (*memcache.Connection, error)

// CachePool re-exposes ResourcePool as a pool of Memcache connection objects.
type CachePool struct {
	name           string
	pool           *pools.ResourcePool
	maxPrefix      sync2.AtomicInt64
	cmd            *exec.Cmd
	rowCacheConfig RowCacheConfig
	capacity       int
	socket         string
	idleTimeout    time.Duration
	memcacheStats  *MemcacheStats
	mu             sync.Mutex
}

// NewCachePool creates a new pool for rowcache connections.
func NewCachePool(name string, rowCacheConfig RowCacheConfig, idleTimeout time.Duration) *CachePool {
	cp := &CachePool{name: name, idleTimeout: idleTimeout}
	if name != "" {
		cp.memcacheStats = NewMemcacheStats(cp, true, false, false)
		stats.Publish(name+"ConnPoolCapacity", stats.IntFunc(cp.Capacity))
		stats.Publish(name+"ConnPoolAvailable", stats.IntFunc(cp.Available))
		stats.Publish(name+"ConnPoolMaxCap", stats.IntFunc(cp.MaxCap))
		stats.Publish(name+"ConnPoolWaitCount", stats.IntFunc(cp.WaitCount))
		stats.Publish(name+"ConnPoolWaitTime", stats.DurationFunc(cp.WaitTime))
		stats.Publish(name+"ConnPoolIdleTimeout", stats.DurationFunc(cp.IdleTimeout))
	}
	http.Handle(statsURL, cp)

	if rowCacheConfig.Binary == "" {
		return cp
	}
	cp.rowCacheConfig = rowCacheConfig

	// Start with memcached defaults
	cp.capacity = 1024 - 50
	if rowCacheConfig.Connections > 0 {
		if rowCacheConfig.Connections <= 50 {
			log.Fatalf("insufficient capacity: %d", rowCacheConfig.Connections)
		}
		cp.capacity = rowCacheConfig.Connections - 50
	}
	return cp
}

// Open opens the pool. It launches memcache and waits till it's up.
func (cp *CachePool) Open() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if cp.pool != nil {
		panic(NewTabletError(ErrFatal, "rowcache is already open"))
	}
	if cp.rowCacheConfig.Binary == "" {
		panic(NewTabletError(ErrFatal, "rowcache binary not specified"))
	}
	cp.socket = generateFilename()
	cp.startMemcache()
	log.Infof("rowcache is enabled")
	f := func() (pools.Resource, error) {
		return memcache.Connect(cp.socket, 10*time.Second)
	}
	cp.pool = pools.NewResourcePool(f, cp.capacity, cp.capacity, cp.idleTimeout)
	if cp.memcacheStats != nil {
		cp.memcacheStats.Open()
	}
}

// generateFilename generates a unique file name. It's convoluted.
// There are race conditions when we have to come up with unique
// names. So, this is a best effort.
func generateFilename() string {
	f, err := ioutil.TempFile("", "mc")
	if err != nil {
		panic(NewTabletError(ErrFatal, "error creating socket file: %v", err))
	}
	name := f.Name()
	err = f.Close()
	if err != nil {
		panic(NewTabletError(ErrFatal, "error closing socket file: %v", err))
	}
	err = os.Remove(name)
	if err != nil {
		panic(NewTabletError(ErrFatal, "error removing socket file: %v", err))
	}
	log.Infof("sock filename: %v", name)
	return name
}

func (cp *CachePool) startMemcache() {
	commandLine := cp.rowCacheConfig.GetSubprocessFlags(cp.socket)
	cp.cmd = exec.Command(commandLine[0], commandLine[1:]...)
	if err := cp.cmd.Start(); err != nil {
		panic(NewTabletError(ErrFatal, "can't start memcache: %v", err))
	}
	attempts := 0
	for {
		time.Sleep(100 * time.Millisecond)
		c, err := memcache.Connect(cp.socket, 30*time.Millisecond)
		if err != nil {
			attempts++
			if attempts >= 50 {
				cp.cmd.Process.Kill()
				// Avoid zombies
				go cp.cmd.Wait()
				// FIXME(sougou): Throw proper error if we can recover
				log.Fatal("Can't connect to memcache")
			}
			continue
		}
		if _, err = c.Set("health", 0, 0, []byte("ok")); err != nil {
			panic(NewTabletError(ErrFatal, "can't communicate with memcache: %v", err))
		}
		c.Close()
		break
	}
}

// Close closes the CachePool. It also shuts down memcache.
// You can call Open again after Close.
func (cp *CachePool) Close() {
	// Close the underlying pool first.
	// You cannot close the pool while holding the
	// lock because we have to still allow Put to
	// return outstanding connections, if any.
	pool := cp.getPool()
	if pool == nil {
		return
	}
	pool.Close()

	// No new operations will be allowed now.
	// Safe to cleanup.
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if cp.pool == nil {
		return
	}
	if cp.memcacheStats != nil {
		cp.memcacheStats.Close()
	}
	cp.cmd.Process.Kill()
	// Avoid zombies
	go cp.cmd.Wait()
	_ = os.Remove(cp.socket)
	cp.socket = ""
	cp.pool = nil
}

// IsClosed returns true if CachePool is closed.
func (cp *CachePool) IsClosed() bool {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	return cp.pool == nil
}

func (cp *CachePool) getPool() *pools.ResourcePool {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	return cp.pool
}

// Get returns a memcache connection from the pool.
// You must call Put after Get.
func (cp *CachePool) Get(ctx context.Context) *memcache.Connection {
	pool := cp.getPool()
	if pool == nil {
		panic(NewTabletError(ErrFatal, "cache pool is not open"))
	}
	r, err := pool.Get(ctx)
	if err != nil {
		panic(NewTabletErrorSql(ErrFatal, err))
	}
	return r.(*memcache.Connection)
}

// Put returns the connection to the pool.
func (cp *CachePool) Put(conn *memcache.Connection) {
	pool := cp.getPool()
	if pool == nil {
		return
	}
	if conn == nil {
		pool.Put(nil)
	} else {
		pool.Put(conn)
	}
}

// StatsJSON returns a JSON version of the CachePool stats.
func (cp *CachePool) StatsJSON() string {
	pool := cp.getPool()
	if pool == nil {
		return "{}"
	}
	return pool.StatsJSON()
}

// Capacity returns the current capacity of the pool.
func (cp *CachePool) Capacity() int64 {
	pool := cp.getPool()
	if pool == nil {
		return 0
	}
	return pool.Capacity()
}

// Available returns the number of available connections in the pool.
func (cp *CachePool) Available() int64 {
	pool := cp.getPool()
	if pool == nil {
		return 0
	}
	return pool.Available()
}

// MaxCap returns the extent to which the pool capacity can be increased.
func (cp *CachePool) MaxCap() int64 {
	pool := cp.getPool()
	if pool == nil {
		return 0
	}
	return pool.MaxCap()
}

// WaitCount returns the number of times we had to wait to get a connection
// from the pool.
func (cp *CachePool) WaitCount() int64 {
	pool := cp.getPool()
	if pool == nil {
		return 0
	}
	return pool.WaitCount()
}

// WaitTime returns the total amount of time spent waiting for a connection.
func (cp *CachePool) WaitTime() time.Duration {
	pool := cp.getPool()
	if pool == nil {
		return 0
	}
	return pool.WaitTime()
}

// IdleTimeout returns the connection idle timeout.
func (cp *CachePool) IdleTimeout() time.Duration {
	pool := cp.getPool()
	if pool == nil {
		return 0
	}
	return pool.IdleTimeout()
}

// ServeHTTP serves memcache stats as HTTP.
func (cp *CachePool) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if err := acl.CheckAccessHTTP(request, acl.MONITORING); err != nil {
		acl.SendError(response, err)
		return
	}
	defer func() {
		if x := recover(); x != nil {
			response.Write(([]byte)(x.(error).Error()))
		}
	}()
	response.Header().Set("Content-Type", "text/plain")
	pool := cp.getPool()
	if pool == nil {
		response.Write(([]byte)("closed"))
		return
	}
	command := request.URL.Path[len(statsURL):]
	if command == "stats" {
		command = ""
	}
	conn := cp.Get(context.Background())
	// This is not the same as defer cp.Put(conn)
	defer func() { cp.Put(conn) }()
	r, err := conn.Stats(command)
	if err != nil {
		conn.Close()
		conn = nil
		response.Write(([]byte)(err.Error()))
	} else {
		response.Write(r)
	}
}
