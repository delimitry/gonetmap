package main

import (
	"flag"
	"log"
	"os"
	"gonetmap"
	"golang.org/x/sys/unix"
	"time"
	"math"
	"fmt"
)

type Eth struct {
	Dst [6]byte
	Src [6]byte
	Type uint16
}

func GetAvail(ring *gonetmap.NmRing) (uint32) {
	if ring.Tail < ring.Cur {
		return ring.Tail - ring.Cur + ring.NumSlots
	} else {
		return ring.Tail - ring.Cur
	}
}

func RingStep(r *gonetmap.NmRing, step uint32) (uint32) {
	cur := r.Cur + step
	if cur >= r.NumSlots {
		cur -= r.NumSlots
	}
	return cur
}

func RingMove(r *gonetmap.NmRing, step uint32) {
	cur := RingStep(r, step)
	r.Cur = cur
	r.Head = cur
}

func ProcessBatch(n *gonetmap.NmDesc, r *gonetmap.NmRing, slots_ptr *[]gonetmap.NmSlot, avail uint32) (uint32) {
	slots := *(slots_ptr)
	for i := uint32(0); i < avail; i++ {
		cur := RingStep(r, i)
		dataPtr := gonetmap.NmBufPtr(r, &slots[cur])
		//data := *gonetmap.NmBufSlicePtr(r, &slots[cur])
		eth := *(*Eth)(dataPtr)
		_ = eth.Src
		_ = eth.Dst
	}
	return avail

}

func main() {
	var (
		iface = flag.String("i", "", "interface")
	)
	flag.Parse()
	if *iface == "" {
		log.Println("usage nm -i netmap:p{0")
		os.Exit(1)
	}
	nm, err := gonetmap.OpenNetmap(*iface)
	if err != nil {
		log.Println(err)
		return
	}

	i := nm.Desc.LastRxRing
	ring := gonetmap.NetmapRing(nm.Desc.NmIf, uint32(i), false)
	slots := gonetmap.GetNmSlots(ring)
	var avail, processed uint32
	cnt := uint64(0)
	ts := time.Now()
	treshold := uint64(100 * math.Pow10(6))
	mult := math.Pow10(3)
	for {
		unix.Poll(nm.Pollset, -1)
		avail = GetAvail(ring)
		processed = ProcessBatch(nm.Desc, ring, slots, avail)
		RingMove(ring, processed)
		cnt += uint64(processed)
		if cnt >= treshold {
			now := time.Now()
			delta := now.UnixNano() - ts.UnixNano()
			rate := float64(cnt) / float64(delta) * mult
			fmt.Printf("%.3f\n", rate)
			ts = now
			cnt = 0
		}
	}
	defer nm.Close()
}