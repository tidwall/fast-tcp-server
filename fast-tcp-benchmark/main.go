package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	var addr string
	var clients int
	var count int
	var pipeline int
	flag.StringVar(&addr, "a", ":3280", "server address")
	flag.IntVar(&clients, "c", 6, "number of clients")
	flag.IntVar(&count, "n", 10000000, "number of messages")
	flag.IntVar(&pipeline, "P", 1, "pipeline messages")
	flag.Parse()

	rand.Seed(time.Now().UnixNano())
	fmt.Printf("generating numbers\n")
	var data []byte // includes linebreaks
	for i := 0; i < count; i++ {
		number := (rand.Int() % (10000000000 - 1000000)) + 1000000
		data = append(data, fmt.Sprintf("%010d\n", number)...)
	}

	fmt.Printf("connecting to server\n")
	var requests int64
	var done int64
	var wg, wg2 sync.WaitGroup
	wg.Add(clients)
	for i := 0; i < clients; i++ {
		var cdata []byte
		if i == clients-1 {
			cdata = data
		} else {
			cdata = data[:count/clients*11]
		}
		data = data[len(cdata):]
		go func(cdata []byte) {
			defer wg.Done()
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				log.Fatal(err)
			}
			defer conn.Close()
			for len(cdata) > 0 {
				if len(cdata) <= pipeline*11 {
					_, err = conn.Write(cdata)
					atomic.AddInt64(&requests, int64(len(cdata)/11))
					cdata = nil
				} else {
					_, err = conn.Write(cdata[:pipeline*11])
					atomic.AddInt64(&requests, int64(pipeline))
					cdata = cdata[pipeline*11:]
				}
				if err != nil {
					log.Fatal(err)
					return
				}
			}
		}(cdata)
	}

	wg2.Add(1)
	go func() {
		start := time.Now()
		for atomic.LoadInt64(&done) == 0 {
			fmt.Printf("\r%.2f", float64(atomic.LoadInt64(&requests))/time.Since(start).Seconds())
			time.Sleep(time.Second / 10)
		}
		fmt.Printf(" requests per second\n")
		wg2.Done()
	}()
	wg.Wait()
	atomic.StoreInt64(&done, 1)
	wg2.Wait()
}
