package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var body []byte
		bodyStr := ""
		sleep := 1

		if r.Body != nil {
			body, _ = ioutil.ReadAll(r.Body)
			if len(body) > 0 {
				bodyStr = string(body)
			}
		}

		if t := r.URL.Query().Get("sleep"); t != "" {
			sleep, _ = strconv.Atoi(t)
		}

		if bodyStr == "" {
			fmt.Printf("Empty body\n")
			return
		}

		topic := ""
		num := 0

		_, err := fmt.Sscanf(bodyStr, "%s %d", &topic, &num)
		if err != nil {
			fmt.Printf("Error parsing: %q %s\n", bodyStr, err)
			return
		}

		// Real work goes here
		time.Sleep(time.Duration(sleep) * time.Second)

		// Report status
		url := "http://knkafkaconsole.default.svc.cluster.local"
		url += fmt.Sprintf("?done=%s", topic)

		for tries := 0; ; tries++ {
			res, err := http.Get(url)
			if err == nil && res != nil && res.StatusCode/100 == 2 {
				break
			}
			// fmt.Printf("Error setting stats: %s %#v\n", err, res)
			if tries == 3 {
				fmt.Printf("Error: Gave up trying to send status: %s\n", err)
				w.WriteHeader(500)
				return
			}
			time.Sleep(time.Duration(tries) * time.Second)
		}
	})

	fmt.Print("Listening on port 8080\n")
	http.ListenAndServe(":8080", nil)
}
