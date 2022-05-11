package main

import (
	"fmt"

	"github.com/blitz-frost/obs"
)

func main() {
	o := obs.NewObserver(100, func(state *int, sample int) {
		*state += sample
		fmt.Println(*state)
	})

	ch := make(chan struct{})
	o.OnFirst = func(state *int, sample int) {
		*state = 44
	}
	o.OnFinal = func() { close(ch) }

	obs.Start(o)

	for i := 0; i < 30; i++ {
		obs.Sample(o, i)
	}

	obs.Stop(o)
	<-ch
}
