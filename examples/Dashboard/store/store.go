package store

import (
	"sync"
	"time"
)

// Transaction represents a single sale or movement.
type Transaction struct {
	ID     string
	Date   string
	Amount float64
	Status string // "Completed", "Pending", "Failed"
	Client string
}

// Stats holds the aggregate metrics for the dashboard overview.
type Stats struct {
	TotalRevenue    float64
	ActiveUsers     int
	ConversionRate  float64
	BounceRate      float64
	RevenueGrowth   float64 // Percentage change
	UsersGrowth     float64 // Percentage change
	ConversionGrowth float64
}

// Store holds the global sales dashboard data.
type Store struct {
	mu           sync.RWMutex
	stats        Stats
	transactions []Transaction
}

// New creates a initialized Store with mock data for the dashboard.
func New() *Store {
	return &Store{
		stats: Stats{
			TotalRevenue:     124500.00,
			ActiveUsers:      8432,
			ConversionRate:   12.5,
			BounceRate:       42.1,
			RevenueGrowth:    14.2,
			UsersGrowth:      5.7,
			ConversionGrowth: -1.2,
		},
		transactions: []Transaction{
			{ID: "TRX-001", Date: time.Now().Format("2006-01-02 15:04"), Amount: 1250.00, Status: "Completed", Client: "Acme Corp"},
			{ID: "TRX-002", Date: time.Now().Add(-2 * time.Hour).Format("2006-01-02 15:04"), Amount: 450.50, Status: "Pending", Client: "Stark Industries"},
			{ID: "TRX-003", Date: time.Now().Add(-5 * time.Hour).Format("2006-01-02 15:04"), Amount: 3200.00, Status: "Completed", Client: "Wayne Enterprises"},
			{ID: "TRX-004", Date: time.Now().Add(-24 * time.Hour).Format("2006-01-02 15:04"), Amount: 150.00, Status: "Failed", Client: "LexCorp"},
			{ID: "TRX-005", Date: time.Now().Add(-30 * time.Hour).Format("2006-01-02 15:04"), Amount: 890.00, Status: "Completed", Client: "Oscorp"},
		},
	}
}

// GetStats returns the current statistics overview.
func (s *Store) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

// GetRecentTransactions returns the latest transactions.
func (s *Store) GetRecentTransactions() []Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]Transaction, len(s.transactions))
	copy(res, s.transactions)
	return res
}
