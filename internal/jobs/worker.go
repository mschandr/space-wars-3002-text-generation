package jobs

import (
	"sync"

	"space-wars-3002-text-generation/internal/config"
	"space-wars-3002-text-generation/internal/db"
	"space-wars-3002-text-generation/internal/dialogue"
	"space-wars-3002-text-generation/internal/llm"
	"space-wars-3002-text-generation/internal/logging"
	"space-wars-3002-text-generation/internal/php"
	"space-wars-3002-text-generation/internal/vendors"
)

// Pool manages a fixed set of worker goroutines that generate dialogue for vendors.
type Pool struct {
	workers      int
	vendorChan   chan vendors.VendorProfile
	wg           sync.WaitGroup
	orchestrator *dialogue.Orchestrator
	logger       *logging.Logger
}

// Result holds the outcome of processing a single vendor.
type Result struct {
	VendorID int64
	Success  bool
	Error    string
}

// NewPool creates a Pool. phpClient may be nil when UseHTTPAPI is false.
func NewPool(database *db.DB, llmClient *llm.Client, phpClient *php.Client, logger *logging.Logger, cfg *config.Config) *Pool {
	return &Pool{
		workers:      cfg.WorkerCount,
		vendorChan:   make(chan vendors.VendorProfile, cfg.WorkerCount),
		orchestrator: dialogue.New(database, llmClient, phpClient, logger, cfg),
		logger:       logger,
	}
}

// Start launches all worker goroutines.
func (p *Pool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Submit sends a vendor to the worker pool for processing.
func (p *Pool) Submit(vendor vendors.VendorProfile) {
	p.vendorChan <- vendor
}

// Close signals no more work and waits for all workers to finish.
func (p *Pool) Close() {
	close(p.vendorChan)
	p.wg.Wait()
}

func (p *Pool) worker(id int) {
	defer p.wg.Done()

	for vendor := range p.vendorChan {
		p.processVendor(id, vendor)
	}
}

func (p *Pool) processVendor(workerID int, vendor vendors.VendorProfile) {
	p.logger.Infof("processing vendor", map[string]interface{}{
		"worker_id":   workerID,
		"vendor_id":   vendor.ID,
		"vendor_uuid": vendor.UUID,
		"service_type": vendor.ServiceType,
	})

	err := p.orchestrator.GenerateForVendor(vendor)
	if err != nil {
		p.logger.Errorf("vendor generation failed", map[string]interface{}{
			"worker_id": workerID,
			"vendor_id": vendor.ID,
			"error":     err.Error(),
		})
		return
	}

	p.logger.Infof("vendor generation complete", map[string]interface{}{
		"worker_id": workerID,
		"vendor_id": vendor.ID,
	})
}
