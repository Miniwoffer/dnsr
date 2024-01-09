package dnsr

import (
	"sync"
	"time"
)

type cache struct {
	capacity    int
	expire      bool
	m           sync.RWMutex
	entries     map[key]entry
	negativeTTL time.Duration
	ttlMax      *time.Duration
}

type key struct {
	qname string
	qtype string
}

type entry struct {
	expiry  time.Time
	rrs RRs
}

const MinCacheCapacity = 1000

// newCache initializes and returns a new cache instance.
// Cache capacity defaults to MinCacheCapacity if <= 0.
func NewCache(capacity int, expire bool, negativeTTL time.Duration, ttlMax *time.Duration) *cache {
	if capacity <= 0 {
		capacity = MinCacheCapacity
	}
	if ttlMax != nil {
		// Make a copy of the ttl
		ttl := *ttlMax
		ttlMax = &ttl
	}


	return &cache{
		capacity: capacity,
		entries:  make(map[key]entry),
		expire:   expire,
		negativeTTL: negativeTTL,
		ttlMax:   ttlMax,
	}
}

// add adds 0 or more DNS records to the resolver cache for a specific
// domain name and record type. This ensures the cache entry exists, even
// if empty, for NXDOMAIN responses.
func (c *cache) add(key key, rrs ...RR) {
	// Prepare the entry
	e := entry{
		rrs:  rrs,
		expiry: time.Now().Add(c.negativeTTL),
	}
	if len(rrs) > 0 {
		e.expiry = rrs[0].Expiry
		for _, rr := range rrs {
			// Its not allowed to mix TTLs in a single responses
			// but if that happens, the lowest TTL should be used
			if rr.Expiry.Before(e.expiry) {
				e.expiry = rr.Expiry
			}
		}
		// Cap the TTL to the maximum allowed
		if c.ttlMax != nil {
			max := time.Now().Add(*c.ttlMax)
			if e.expiry.After(max) {
				e.expiry = max
			}
		}
	}
	c.m.Lock()
	defer c.m.Unlock()
	c._add(key,e)
}

// _add does NOT lock the mutex so unsafe for concurrent usage.
func (c *cache) _add(key key, entry entry) {
	if len(c.entries) >= c.capacity {
		c._evict()
	}
	c.entries[key] = entry
}

// FIXME: better random cache eviction than Go’s random key guarantee?
// Not safe for concurrent usage.
func (c *cache) _evict() {

	if len(c.entries) < c.capacity {
		return
	}
	// First evict expired entries
	if c.expire {
		now := time.Now()
		for k, e := range c.entries {
			// Remove expired entries and entries with expiration
			if e.expiry.Before(now) {
				delete(c.entries, k)
			}
			if len(c.entries) < c.capacity {
				return
			}
		}
	}

	// Then randomly evict entries
	for k := range c.entries {
		delete(c.entries, k)
		if len(c.entries) < c.capacity {
			return
		}
	}
}

// get returns a randomly ordered slice of DNS records.
func (c *cache) get(key key) RRs {
	c.m.RLock()
	defer c.m.RUnlock()
	e, ok := c.entries[key]
	if !ok {
		return nil
	}
	if !c.expire || e.expiry.After(time.Now()) {
		return e.rrs
	}
	return nil
}
