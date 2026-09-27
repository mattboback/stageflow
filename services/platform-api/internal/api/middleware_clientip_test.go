package api

import (
	"sync"
)

func resetTrustedProxiesForTest() {
	trustedProxiesOnce = sync.Once{}
	trustedProxiesVal = nil
}
