package timing

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestTicker(t *testing.T) {
	log.SetFlags(log.Ldate | log.Ltime)
	tm := NewTickerManager(time.Second, time.Millisecond*100, true, func(tk Ticker) {
		fmt.Printf("%+v %+v\n", time.Now().Format("15:04:05.000"), tk.Value())
	})

	tm.Start()

	for i := 0; i < 20; i++ {
		tm.Create("", i)
	}

	time.Sleep(time.Second * 2)
	tm.Stop()
	t.Log("stoped")
}
