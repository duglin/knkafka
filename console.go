package main

import (
	"fmt"
	"net/http"
	"sync"
)

func main() {
	stats := map[string]int{} // topic->count
	totalDone := 0
	valMutex := sync.Mutex{}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("clear") != "" {
			valMutex.Lock()
			stats = map[string]int{}
			totalDone = 0
			valMutex.Unlock()
			return
		}

		if topic := r.URL.Query().Get("done"); topic != "" {
			// done=TOPIC
			val := 0
			valMutex.Lock()
			if t, ok := stats[topic]; ok {
				val = t
			}
			stats[topic] = val + 1
			totalDone++
			valMutex.Unlock()
			return
		}

		valMutex.Lock()
		response := fmt.Sprintf("Total done: %d\n", totalDone)
		for topic, val := range stats {
			response += fmt.Sprintf(" %s : %d\n", topic, val)
		}
		valMutex.Unlock()

		w.Write([]byte(response))
	})

	fmt.Print("Listening on port 8080\n")
	http.ListenAndServe(":8080", nil)
}
