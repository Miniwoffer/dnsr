package dnsr

import (
	"sync"
	"testing"
	"time"

	"github.com/nbio/st"
)

func TestCache(t *testing.T) {
	c := NewCache(100, false, 0, nil)

	rr := RR{Name: "hello.", Type: "A", Value: "1.2.3.4"}
	key := key{rr.Name, rr.Type}
	c.add(key)
	c.add(key, rr)
	rrs := c.get(key)
	st.Expect(t, len(rrs), 1)
}

func TestLiveCacheEntry(t *testing.T) {
	c := NewCache(100, true, 0, nil)

	alive := time.Now().Add(time.Minute)
	rr := RR{Name: "alive.", Type: "A", Value: "1.2.3.4", Expiry: alive}
	key := key{rr.Name, rr.Type}
	c.add(key)
	c.add(key, rr)
	rrs := c.get(key)
	st.Expect(t, len(rrs), 1)
}

func TestExpiredCacheEntry(t *testing.T) {
	c := NewCache(100, true, 0, nil)

	expired := time.Now().Add(-time.Minute)
	rr := RR{Name: "expired.", Type: "A", Value: "1.2.3.4", Expiry: expired}
	key := key{rr.Name, rr.Type}
	c.add(key)

	c.add(key, rr)
	rrs := c.get(key)
	st.Expect(t, len(rrs), 0)
}

func TestCacheContention(t *testing.T) {
	k := "expired."
	key := key{k, "A"}
	c := NewCache(10, true, 0, nil)
	var wg sync.WaitGroup
	f := func() {
		rrs := c.get(key)
		st.Expect(t, len(rrs), 0)
		c.add(key)
		expired := time.Now().Add(-time.Minute)
		rr := RR{Name: k, Type: "A", Value: "1.2.3.4", Expiry: expired}
		c.add(key, rr)
		wg.Done()
	}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go f()
	}
	wg.Wait()
}
