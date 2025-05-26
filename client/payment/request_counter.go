// client/payment/request_counter.go

package payment

import (
    "fmt"
    "sync"
    "encoding/json"
    "io/ioutil"
    "os"
    "path/filepath"
)

// RequestCounter manages unique request IDs per client
type RequestCounter struct {
    counters    map[string]int
    counterFile string
    mutex       sync.Mutex
}

// NewRequestCounter creates a new request counter
func NewRequestCounter(counterFile string) *RequestCounter {
    rc := &RequestCounter{
        counters:    make(map[string]int),
        counterFile: counterFile,
    }
    
    // Ensure directory exists
    dir := filepath.Dir(counterFile)
    if _, err := os.Stat(dir); os.IsNotExist(err) {
        os.MkdirAll(dir, 0755)
    }
    
    // Load existing counters
    rc.loadCounters()
    
    return rc
}

// GetNextRequestID returns the next request ID for a client and increments the counter
func (rc *RequestCounter) GetNextRequestID(clientName string) string {
    rc.mutex.Lock()
    defer rc.mutex.Unlock()
    
    counter := rc.counters[clientName]
    requestID := fmt.Sprintf("%s_req_%d", clientName, counter)
    
    return requestID
}

// IncrementCounter should be called after a successful payment
func (rc *RequestCounter) IncrementCounter(clientName string) {
    rc.mutex.Lock()
    defer rc.mutex.Unlock()
    
    rc.counters[clientName]++
    rc.saveCounters()
}

// loadCounters loads counters from file
func (rc *RequestCounter) loadCounters() {
    data, err := ioutil.ReadFile(rc.counterFile)
    if err != nil {
        return // File might not exist yet
    }
    
    json.Unmarshal(data, &rc.counters)
}

// saveCounters saves counters to file
func (rc *RequestCounter) saveCounters() {
    data, err := json.Marshal(rc.counters)
    if err != nil {
        return
    }
    
    ioutil.WriteFile(rc.counterFile, data, 0644)
}