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
	TotalRevenue     float64
	ActiveUsers      int
	ConversionRate   float64
	BounceRate       float64
	RevenueGrowth    float64 // Percentage change
	UsersGrowth      float64 // Percentage change
	ConversionGrowth float64
}

// SalesRecord represents an individual sale with product details.
type SalesRecord struct {
	ID       string
	Date     string
	Product  string
	Category string
	Quantity int
	UnitPrice float64
	Total    float64
	Region   string
	Rep      string
	Status   string // "Closed", "Negotiating", "Lost"
}

// SalesSummary holds aggregate sales metrics.
type SalesSummary struct {
	TotalSales      float64
	TotalOrders     int
	AvgOrderValue   float64
	TopProduct      string
	SalesGrowth     float64
	OrdersGrowth    float64
	MonthlySales    []MonthSales
}

// MonthSales represents sales for a single month.
type MonthSales struct {
	Month  string
	Amount float64
}

// Customer represents a customer record.
type Customer struct {
	ID        string
	Name      string
	Email     string
	Company   string
	Plan      string // "Enterprise", "Pro", "Starter"
	Spent     float64
	Orders    int
	JoinDate  string
	Status    string // "Active", "Inactive", "Churned"
}

// CustomerSummary holds aggregate customer metrics.
type CustomerSummary struct {
	TotalCustomers   int
	ActiveCustomers  int
	ChurnRate        float64
	AvgLifetimeValue float64
	NewThisMonth     int
	CustomerGrowth   float64
	RetentionRate    float64
}

// Store holds the global sales dashboard data.
type Store struct {
	mu           sync.RWMutex
	stats        Stats
	transactions []Transaction
	sales        []SalesRecord
	salesSummary SalesSummary
	customers    []Customer
	customerSummary CustomerSummary
}

// New creates a initialized Store with mock data for the dashboard.
func New() *Store {
	now := time.Now()
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
			{ID: "TRX-001", Date: now.Format("2006-01-02 15:04"), Amount: 1250.00, Status: "Completed", Client: "Acme Corp"},
			{ID: "TRX-002", Date: now.Add(-2 * time.Hour).Format("2006-01-02 15:04"), Amount: 450.50, Status: "Pending", Client: "Stark Industries"},
			{ID: "TRX-003", Date: now.Add(-5 * time.Hour).Format("2006-01-02 15:04"), Amount: 3200.00, Status: "Completed", Client: "Wayne Enterprises"},
			{ID: "TRX-004", Date: now.Add(-24 * time.Hour).Format("2006-01-02 15:04"), Amount: 150.00, Status: "Failed", Client: "LexCorp"},
			{ID: "TRX-005", Date: now.Add(-30 * time.Hour).Format("2006-01-02 15:04"), Amount: 890.00, Status: "Completed", Client: "Oscorp"},
		},
		sales: []SalesRecord{
			{ID: "SAL-001", Date: now.Format("2006-01-02"), Product: "Thunder Enterprise", Category: "Software", Quantity: 3, UnitPrice: 12000.00, Total: 36000.00, Region: "North America", Rep: "Alice Johnson", Status: "Closed"},
			{ID: "SAL-002", Date: now.Add(-24 * time.Hour).Format("2006-01-02"), Product: "Thunder Pro", Category: "Software", Quantity: 10, UnitPrice: 2400.00, Total: 24000.00, Region: "Europe", Rep: "Bob Smith", Status: "Closed"},
			{ID: "SAL-003", Date: now.Add(-48 * time.Hour).Format("2006-01-02"), Product: "Consulting Pack", Category: "Services", Quantity: 1, UnitPrice: 18500.00, Total: 18500.00, Region: "North America", Rep: "Alice Johnson", Status: "Closed"},
			{ID: "SAL-004", Date: now.Add(-72 * time.Hour).Format("2006-01-02"), Product: "Thunder Enterprise", Category: "Software", Quantity: 1, UnitPrice: 12000.00, Total: 12000.00, Region: "Asia Pacific", Rep: "Carla Reyes", Status: "Negotiating"},
			{ID: "SAL-005", Date: now.Add(-96 * time.Hour).Format("2006-01-02"), Product: "Thunder Starter", Category: "Software", Quantity: 25, UnitPrice: 480.00, Total: 12000.00, Region: "Europe", Rep: "Bob Smith", Status: "Closed"},
			{ID: "SAL-006", Date: now.Add(-120 * time.Hour).Format("2006-01-02"), Product: "Training Program", Category: "Services", Quantity: 2, UnitPrice: 4500.00, Total: 9000.00, Region: "Latin America", Rep: "Diego Morales", Status: "Closed"},
			{ID: "SAL-007", Date: now.Add(-144 * time.Hour).Format("2006-01-02"), Product: "Thunder Pro", Category: "Software", Quantity: 5, UnitPrice: 2400.00, Total: 12000.00, Region: "North America", Rep: "Alice Johnson", Status: "Negotiating"},
			{ID: "SAL-008", Date: now.Add(-168 * time.Hour).Format("2006-01-02"), Product: "Thunder Enterprise", Category: "Software", Quantity: 2, UnitPrice: 12000.00, Total: 24000.00, Region: "Europe", Rep: "Eva Klein", Status: "Lost"},
		},
		salesSummary: SalesSummary{
			TotalSales:    147500.00,
			TotalOrders:   156,
			AvgOrderValue: 945.51,
			TopProduct:    "Thunder Enterprise",
			SalesGrowth:   18.3,
			OrdersGrowth:  7.2,
			MonthlySales: []MonthSales{
				{Month: "Oct", Amount: 98000},
				{Month: "Nov", Amount: 112000},
				{Month: "Dec", Amount: 105000},
				{Month: "Jan", Amount: 124500},
				{Month: "Feb", Amount: 131000},
				{Month: "Mar", Amount: 147500},
			},
		},
		customers: []Customer{
			{ID: "CUS-001", Name: "Tony Stark", Email: "tony@stark.com", Company: "Stark Industries", Plan: "Enterprise", Spent: 84500.00, Orders: 23, JoinDate: "2024-01-15", Status: "Active"},
			{ID: "CUS-002", Name: "Bruce Wayne", Email: "bruce@wayne.com", Company: "Wayne Enterprises", Plan: "Enterprise", Spent: 72000.00, Orders: 18, JoinDate: "2024-02-20", Status: "Active"},
			{ID: "CUS-003", Name: "Pepper Potts", Email: "pepper@stark.com", Company: "Stark Industries", Plan: "Pro", Spent: 34200.00, Orders: 12, JoinDate: "2024-03-10", Status: "Active"},
			{ID: "CUS-004", Name: "Lex Luthor", Email: "lex@lexcorp.com", Company: "LexCorp", Plan: "Pro", Spent: 28900.00, Orders: 9, JoinDate: "2024-04-05", Status: "Inactive"},
			{ID: "CUS-005", Name: "Norman Osborn", Email: "norman@oscorp.com", Company: "Oscorp", Plan: "Starter", Spent: 12400.00, Orders: 5, JoinDate: "2024-05-18", Status: "Churned"},
			{ID: "CUS-006", Name: "Diana Prince", Email: "diana@themyscira.io", Company: "Themyscira Inc", Plan: "Enterprise", Spent: 96000.00, Orders: 31, JoinDate: "2023-11-01", Status: "Active"},
			{ID: "CUS-007", Name: "Clark Kent", Email: "clark@dailyplanet.com", Company: "Daily Planet", Plan: "Starter", Spent: 8600.00, Orders: 4, JoinDate: "2024-06-22", Status: "Active"},
			{ID: "CUS-008", Name: "Barry Allen", Email: "barry@starlabs.com", Company: "S.T.A.R. Labs", Plan: "Pro", Spent: 41500.00, Orders: 15, JoinDate: "2024-01-30", Status: "Active"},
		},
		customerSummary: CustomerSummary{
			TotalCustomers:   243,
			ActiveCustomers:  198,
			ChurnRate:        4.2,
			AvgLifetimeValue: 12450.00,
			NewThisMonth:     18,
			CustomerGrowth:   8.5,
			RetentionRate:    95.8,
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

// GetSalesRecords returns all sales records.
func (s *Store) GetSalesRecords() []SalesRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]SalesRecord, len(s.sales))
	copy(res, s.sales)
	return res
}

// GetSalesSummary returns the sales summary metrics.
func (s *Store) GetSalesSummary() SalesSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.salesSummary
}

// GetCustomers returns all customer records.
func (s *Store) GetCustomers() []Customer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]Customer, len(s.customers))
	copy(res, s.customers)
	return res
}

// GetCustomerSummary returns customer aggregate metrics.
func (s *Store) GetCustomerSummary() CustomerSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.customerSummary
}
