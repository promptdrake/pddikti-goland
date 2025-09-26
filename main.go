package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"

	"backendsaveapi/route"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

var (
	serverStart     = time.Now()
	totalRequests   uint64
	totalErrors     uint64
	inFlight        int64
	totalLatencyNS  uint64
	lastRequestUnix int64
)

// ✅ Define response struct
type statResponse struct {
	UptimeSeconds       float64 `json:"uptime_seconds"`
	RequestsTotal       uint64  `json:"requests_total"`
	ErrorsTotal         uint64  `json:"errors_total"`
	RequestsInFlight    int64   `json:"requests_in_flight"`
	AvgLatencyMs        float64 `json:"avg_latency_ms"`
	ReqPerSecond        float64 `json:"req_per_second"`
	LastRequestUnix     int64   `json:"last_request_unix"`
	GoRoutines          int     `json:"go_routines"`
	MemAllocBytes       uint64  `json:"mem_alloc_bytes"`
	MemSysBytes         uint64  `json:"mem_sys_bytes"`
	MemHeapAllocBytes   uint64  `json:"mem_heap_alloc_bytes"`
	MemHeapSysBytes     uint64  `json:"mem_heap_sys_bytes"`
	DBOpenConnections   int     `json:"db_open_connections"`
	DBInUse             int     `json:"db_in_use"`
	DBIdle              int     `json:"db_idle"`
	DBWaitCount         int64   `json:"db_wait_count"`
	DBWaitDurationMs    int64   `json:"db_wait_duration_ms"`
	DBMaxIdleClosed     int64   `json:"db_max_idle_closed"`
	DBMaxLifetimeClosed int64   `json:"db_max_lifetime_closed"`
	DBMaxIdleTimeClosed int64   `json:"db_max_idle_time_closed"`
}

func main() {
	// Load .env if available
	_ = godotenv.Load()

	router := mux.NewRouter()
	router.HandleFunc("/", handleempat).Methods("GET")
	router.HandleFunc("/carimahasiswa", route.CariMahasiswa).Methods("GET")
	// ✅ Register stats endpoint
	router.HandleFunc("/stats", statServerHandler).Methods("GET")

	loggedRouter := loggingMiddleware(router)

	fmt.Println("Server running on :9040")
	log.Fatal(http.ListenAndServe(":9040", loggedRouter))
}

func handleempat(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, "mas ipan gamtenk amjir :V")
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(b []byte) (int, error) {
	if sr.status == 0 {
		sr.status = http.StatusOK
	}
	return sr.ResponseWriter.Write(b)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		atomic.AddInt64(&inFlight, 1)

		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)

		dur := time.Since(start)
		atomic.AddUint64(&totalRequests, 1)
		atomic.AddUint64(&totalLatencyNS, uint64(dur.Nanoseconds()))
		atomic.StoreInt64(&lastRequestUnix, time.Now().Unix())

		if rec.status >= 400 {
			atomic.AddUint64(&totalErrors, 1)
		}

		atomic.AddInt64(&inFlight, -1)

		fmt.Printf("%dms %s %s -> %d\n", dur.Milliseconds(), r.Method, r.URL.Path, rec.status)
	})
}

func statServerHandler(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(serverStart).Seconds()
	reqs := atomic.LoadUint64(&totalRequests)
	errs := atomic.LoadUint64(&totalErrors)
	infl := atomic.LoadInt64(&inFlight)
	totalNS := atomic.LoadUint64(&totalLatencyNS)
	lastReq := atomic.LoadInt64(&lastRequestUnix)

	var avgMs float64
	if reqs > 0 {
		avgMs = float64(totalNS) / 1e6 / float64(reqs)
	}

	var rps float64
	if uptime > 0 {
		rps = float64(reqs) / uptime
	}

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	resp := statResponse{
		UptimeSeconds:     uptime,
		RequestsTotal:     reqs,
		ErrorsTotal:       errs,
		RequestsInFlight:  infl,
		AvgLatencyMs:      avgMs,
		ReqPerSecond:      rps,
		LastRequestUnix:   lastReq,
		GoRoutines:        runtime.NumGoroutine(),
		MemAllocBytes:     ms.Alloc,
		MemSysBytes:       ms.Sys,
		MemHeapAllocBytes: ms.HeapAlloc,
		MemHeapSysBytes:   ms.HeapSys,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
