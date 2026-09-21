// A tiny uptime checker. POST /check a JSON list of URLs; the service
// probes each one and reports whether it's up and how long it took.
// A probe is slow (a network round-trip) and some hosts hang. Try a few
// curl batches before you touch anything.
//
// 1.  **A big batch must not melt the box.**
//     - A /check with thousands of URLs must keep the number of probes
//       running at once bounded, no matter how long the list is.
//
// 2.  **Every URL gets exactly one trustworthy result.**
//     - Hammer it with -race. The response must carry one correct result
//       per input URL, with nothing lost or corrupted under load.
//
// 3.  **If the caller hangs up, stop probing.**
//     - When the client disconnects mid-batch, in-flight and not-yet-
//       started probes should stop rather than run to completion.
//
// 4.  **One stuck host must not hold the whole batch hostage.**
//     - A single hanging URL must not make the caller wait on it. Bound
//       how long any one probe may take and report the slow one as down.

package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type CheckRequest struct {
	URLs []string `json:"urls"`
}

type Result struct {
	URL string `json:"url"`
	Up  bool   `json:"up"`
	Ms  int64  `json:"ms"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/check", checkHandler)
	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func checkHandler(w http.ResponseWriter, r *http.Request) {
	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	results := map[string]Result{}

	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, u := range req.URLs {
		wg.Add(1)
		go func(u string) {
                        mu.Lock();
                        res := probe(u);
			defer mu.Unlock()
			defer wg.Done()
			results[u] = res
		}(u)
	}
	wg.Wait()

	out := make([]Result, 0, len(results))
	for _, res := range results {
		out = append(out, res)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string][]Result{"results": out})
}

// probe simulates a health check. Most hosts answer quickly; some hang.
func probe(u string) Result {
	start := time.Now()
	time.Sleep(time.Duration(50+rand.Intn(350)) * time.Millisecond)
	if rand.Intn(8) == 0 {
		time.Sleep(5 * time.Second) // a stuck host
	}
	return Result{URL: u, Up: rand.Intn(5) != 0, Ms: time.Since(start).Milliseconds()}
}
