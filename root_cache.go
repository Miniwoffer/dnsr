package dnsr

import (
	"strings"

	"github.com/miekg/dns"
)

//go:generate sh generate.sh

var (
	rootCache *cache
)

func init() {
	rootCache = NewCache(strings.Count(root, "\n"), false, 0, nil)
	zp := dns.NewZoneParser(strings.NewReader(root), "", "")

	for drr, ok := zp.Next(); ok; drr, ok = zp.Next() {
		rr, ok := convertRR(drr, false)
		if ok {
			rootCache.add(key{rr.Name, rr.Type}, rr)
		}
	}

	if err := zp.Err(); err != nil {
		panic(err)
	}
}
