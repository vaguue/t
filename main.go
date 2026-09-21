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
        var wg sync.WaitGroup

        var mu sync.Mutex

        s := make(chan struct{}, 5)

        select {
          case <-s:
            wg.Add(1)
            go func(u string) {
                    defer wg.Done()
                    res := probe(u)
                    mu.Lock()
                    defer mu.Unlock()
                    results[u] = res
            }(u)
            return
        }

        for _, u := range req.URLs {

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
