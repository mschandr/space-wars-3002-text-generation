package dialogue

import (
	"fmt"
	"time"

	"space-wars-3002-text-generation/internal/config"
	"space-wars-3002-text-generation/internal/constants"
	"space-wars-3002-text-generation/internal/db"
	"space-wars-3002-text-generation/internal/llm"
	"space-wars-3002-text-generation/internal/logging"
	"space-wars-3002-text-generation/internal/php"
	"space-wars-3002-text-generation/internal/prompts"
	"space-wars-3002-text-generation/internal/validation"
	"space-wars-3002-text-generation/internal/vendors"
)

// DialogueBucket represents one scope in the generation matrix.
type DialogueBucket struct {
	LineType           string
	Bucket             string
	TransactionContext string
	InventoryContext   string
	RequestedCount     int
}

// Orchestrator drives generation for a single vendor across all dialogue scopes.
type Orchestrator struct {
	db        *db.DB
	llm       *llm.Client
	phpClient *php.Client // nil when UseHTTPAPI is false
	logger    *logging.Logger
	cfg       *config.Config
}

func New(database *db.DB, llmClient *llm.Client, phpClient *php.Client, logger *logging.Logger, cfg *config.Config) *Orchestrator {
	return &Orchestrator{
		db:        database,
		llm:       llmClient,
		phpClient: phpClient,
		logger:    logger,
		cfg:       cfg,
	}
}

// GenerateForVendor runs the full generation matrix for one vendor.
// Marks the vendor generating before starting and complete/failed when done.
func (o *Orchestrator) GenerateForVendor(vendor vendors.VendorProfile) error {
	if err := o.markGenerating(vendor); err != nil {
		// Log but don't abort — status tracking is best-effort
		o.logger.Warnf("failed to mark vendor as generating", map[string]interface{}{
			"vendor_id":   vendor.ID,
			"vendor_uuid": vendor.UUID,
			"error":       err.Error(),
		})
	}

	buckets := getDialogueBuckets()
	failedBuckets := 0

	for _, bucket := range buckets {
		if err := o.generateBucket(vendor, bucket); err != nil {
			o.logger.Errorf("bucket failed", map[string]interface{}{
				"vendor_id":           vendor.ID,
				"line_type":           bucket.LineType,
				"bucket":              bucket.Bucket,
				"transaction_context": bucket.TransactionContext,
				"inventory_context":   bucket.InventoryContext,
				"error":               err.Error(),
			})
			failedBuckets++
			// Continue to next bucket — partial generation is better than none.
		}
	}

	if failedBuckets > 0 {
		_ = o.markFailed(vendor)
		return fmt.Errorf("vendor %d: %d/%d buckets failed", vendor.ID, failedBuckets, len(buckets))
	}

	return o.markComplete(vendor)
}

// generateBucket generates and stores lines for one dialogue scope.
func (o *Orchestrator) generateBucket(vendor vendors.VendorProfile, bucket DialogueBucket) error {
	o.logger.Infof("generating bucket", map[string]interface{}{
		"vendor_id":           vendor.ID,
		"vendor_uuid":         vendor.UUID,
		"line_type":           bucket.LineType,
		"bucket":              bucket.Bucket,
		"transaction_context": bucket.TransactionContext,
		"inventory_context":   bucket.InventoryContext,
		"requested_count":     bucket.RequestedCount,
	})

	var lastErr error
	requestedCount := bucket.RequestedCount

	for attempt := 0; attempt <= o.cfg.GenerationRetryMax; attempt++ {
		lines, promptHash, err := o.callLLM(vendor, bucket, requestedCount, attempt)
		if err != nil {
			o.logger.Infof("LLM call failed", map[string]interface{}{
				"vendor_id":   vendor.ID,
				"attempt":     attempt,
				"error":       err.Error(),
			})
			lastErr = err
			requestedCount = max(3, int(float64(requestedCount)*0.8))
			continue
		}

		accepted, rejected, err := validation.Validate(lines)
		if err != nil {
			o.logger.Infof("validation failed, will retry", map[string]interface{}{
				"vendor_id":   vendor.ID,
				"attempt":     attempt,
				"accepted":    len(accepted),
				"rejected":    len(rejected),
				"error":       err.Error(),
			})
			lastErr = err
			requestedCount = max(3, int(float64(requestedCount)*0.8))
			continue
		}

		o.logger.Infof("lines validated", map[string]interface{}{
			"vendor_id":   vendor.ID,
			"accepted":    len(accepted),
			"rejected":    len(rejected),
			"prompt_hash": promptHash,
			"retry_count": attempt,
		})

		if !o.cfg.DryRun {
			if err := o.storeLines(vendor, bucket, accepted); err != nil {
				return fmt.Errorf("failed to store lines: %w", err)
			}
		}

		o.logger.Infof("bucket complete", map[string]interface{}{
			"vendor_id":           vendor.ID,
			"line_type":           bucket.LineType,
			"transaction_context": bucket.TransactionContext,
			"inventory_context":   bucket.InventoryContext,
			"inserted":            len(accepted),
		})

		return nil
	}

	return lastErr
}

// callLLM builds a prompt and calls the LLM. Returns lines, prompt hash, and any error.
func (o *Orchestrator) callLLM(vendor vendors.VendorProfile, bucket DialogueBucket, count, attempt int) ([]string, string, error) {
	p := prompts.BuildPrompt(
		vendor,
		bucket.LineType,
		bucket.Bucket,
		bucket.TransactionContext,
		bucket.InventoryContext,
		count,
	)

	o.logger.Debugf("calling LLM", map[string]interface{}{
		"vendor_id":   vendor.ID,
		"prompt_hash": p.Hash,
		"attempt":     attempt,
	})

	if o.cfg.DryRun {
		o.logger.Infof("dry run: skipping LLM call", map[string]interface{}{
			"vendor_id": vendor.ID,
		})
		return nil, p.Hash, nil
	}

	lines, err := o.llm.Generate(p.System, p.User, attempt)
	return lines, p.Hash, err
}

// storeLines routes to HTTP or direct DB storage based on config.
func (o *Orchestrator) storeLines(vendor vendors.VendorProfile, bucket DialogueBucket, lines []string) error {
	if o.cfg.UseHTTPAPI {
		return o.storeLinesViaHTTP(vendor, bucket, lines)
	}
	return o.storeLinesDirectDB(vendor, bucket, lines)
}

// storeLinesViaHTTP submits lines to PHP's internal API for validation and storage.
func (o *Orchestrator) storeLinesViaHTTP(vendor vendors.VendorProfile, bucket DialogueBucket, lines []string) error {
	req := php.SubmitLinesRequest{
		LineType:           bucket.LineType,
		InteractionBucket:  bucket.Bucket,
		TransactionContext: bucket.TransactionContext,
		InventoryContext:   bucket.InventoryContext,
		GenerationVersion:  vendor.DialogueGenerationVersion,
		Lines:              lines,
	}
	return o.phpClient.SubmitLines(vendor.UUID, req)
}

// storeLinesDirectDB deletes old rows for the exact scope and inserts fresh ones.
func (o *Orchestrator) storeLinesDirectDB(vendor vendors.VendorProfile, bucket DialogueBucket, lines []string) error {
	_, err := o.db.Exec(`
		DELETE FROM vendor_dialogue
		WHERE galaxy_vendor_profile_id = ?
		  AND line_type = ?
		  AND interaction_bucket = ?
		  AND transaction_context = ?
		  AND inventory_context = ?
	`, vendor.ID, bucket.LineType, bucket.Bucket, bucket.TransactionContext, bucket.InventoryContext)
	if err != nil {
		return fmt.Errorf("failed to delete old dialogue rows: %w", err)
	}

	for _, line := range lines {
		_, err := o.db.Exec(`
			INSERT INTO vendor_dialogue
				(galaxy_vendor_profile_id, line_type, interaction_bucket, transaction_context,
				 inventory_context, line_text, weight, generation_version, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, 1.0000, ?, NOW(), NOW())
		`, vendor.ID, bucket.LineType, bucket.Bucket, bucket.TransactionContext,
			bucket.InventoryContext, line, vendor.DialogueGenerationVersion)
		if err != nil {
			return fmt.Errorf("failed to insert dialogue line: %w", err)
		}
	}

	return nil
}

// markGenerating updates vendor status to "generating".
func (o *Orchestrator) markGenerating(vendor vendors.VendorProfile) error {
	if o.cfg.UseHTTPAPI {
		return o.phpClient.UpdateStatus(vendor.UUID, constants.StatusGenerating, "")
	}
	return vendors.MarkGenerating(o.db, vendor.ID)
}

// markComplete updates vendor status to "complete" with the current timestamp.
func (o *Orchestrator) markComplete(vendor vendors.VendorProfile) error {
	if o.cfg.UseHTTPAPI {
		generatedAt := time.Now().UTC().Format(time.RFC3339)
		return o.phpClient.UpdateStatus(vendor.UUID, constants.StatusComplete, generatedAt)
	}
	return vendors.MarkComplete(o.db, vendor.ID)
}

// markFailed updates vendor status to "failed".
func (o *Orchestrator) markFailed(vendor vendors.VendorProfile) error {
	if o.cfg.UseHTTPAPI {
		return o.phpClient.UpdateStatus(vendor.UUID, constants.StatusFailed, "")
	}
	return vendors.MarkFailed(o.db, vendor.ID)
}

// getDialogueBuckets returns the full v1 generation matrix (22 scopes).
// Every entry is a distinct (lineType, bucket, transactionContext, inventoryContext) combination.
func getDialogueBuckets() []DialogueBucket {
	return []DialogueBucket{
		// Greetings — neutral, no inventory context
		{LineType: constants.LineTypeGreeting, Bucket: constants.BucketFirstVisit, TransactionContext: constants.TxNeutral, InventoryContext: constants.InvNone, RequestedCount: 15},
		{LineType: constants.LineTypeGreeting, Bucket: constants.BucketSecondVisit, TransactionContext: constants.TxNeutral, InventoryContext: constants.InvNone, RequestedCount: 15},
		{LineType: constants.LineTypeGreeting, Bucket: constants.BucketThirdVisit, TransactionContext: constants.TxNeutral, InventoryContext: constants.InvNone, RequestedCount: 15},
		{LineType: constants.LineTypeGreeting, Bucket: constants.BucketRepeatCustomer, TransactionContext: constants.TxNeutral, InventoryContext: constants.InvNone, RequestedCount: 15},

		// Inventory pitches — vendor selling by item category
		{LineType: constants.LineTypeInventoryPitch, Bucket: constants.BucketRepeatCustomer, TransactionContext: constants.TxVendorSelling, InventoryContext: constants.InvShip, RequestedCount: 10},
		{LineType: constants.LineTypeInventoryPitch, Bucket: constants.BucketRepeatCustomer, TransactionContext: constants.TxVendorSelling, InventoryContext: constants.InvShieldProjector, RequestedCount: 10},
		{LineType: constants.LineTypeInventoryPitch, Bucket: constants.BucketRepeatCustomer, TransactionContext: constants.TxVendorSelling, InventoryContext: constants.InvEngine, RequestedCount: 10},
		{LineType: constants.LineTypeInventoryPitch, Bucket: constants.BucketRepeatCustomer, TransactionContext: constants.TxVendorSelling, InventoryContext: constants.InvReactor, RequestedCount: 10},
		{LineType: constants.LineTypeInventoryPitch, Bucket: constants.BucketRepeatCustomer, TransactionContext: constants.TxVendorSelling, InventoryContext: constants.InvWeapon, RequestedCount: 10},
		{LineType: constants.LineTypeInventoryPitch, Bucket: constants.BucketRepeatCustomer, TransactionContext: constants.TxVendorSelling, InventoryContext: constants.InvSensorArray, RequestedCount: 10},
		{LineType: constants.LineTypeInventoryPitch, Bucket: constants.BucketRepeatCustomer, TransactionContext: constants.TxVendorSelling, InventoryContext: constants.InvCargoModule, RequestedCount: 10},
		{LineType: constants.LineTypeInventoryPitch, Bucket: constants.BucketRepeatCustomer, TransactionContext: constants.TxVendorSelling, InventoryContext: constants.InvHullPlating, RequestedCount: 10},
		{LineType: constants.LineTypeInventoryPitch, Bucket: constants.BucketRepeatCustomer, TransactionContext: constants.TxVendorSelling, InventoryContext: constants.InvSalvageComponent, RequestedCount: 10},

		// Inventory pitches — vendor buying (skeptical/appraising tone)
		{LineType: constants.LineTypeInventoryPitch, Bucket: constants.BucketRepeatCustomer, TransactionContext: constants.TxVendorBuying, InventoryContext: constants.InvShip, RequestedCount: 10},
		{LineType: constants.LineTypeInventoryPitch, Bucket: constants.BucketRepeatCustomer, TransactionContext: constants.TxVendorBuying, InventoryContext: constants.InvEngine, RequestedCount: 10},
		{LineType: constants.LineTypeInventoryPitch, Bucket: constants.BucketRepeatCustomer, TransactionContext: constants.TxVendorBuying, InventoryContext: constants.InvSalvageComponent, RequestedCount: 10},

		// Deal outcomes
		{LineType: constants.LineTypeDealAccepted, Bucket: constants.BucketRepeatCustomer, TransactionContext: constants.TxVendorSelling, InventoryContext: constants.InvNone, RequestedCount: 10},
		{LineType: constants.LineTypeDealAccepted, Bucket: constants.BucketRepeatCustomer, TransactionContext: constants.TxVendorBuying, InventoryContext: constants.InvNone, RequestedCount: 10},
		{LineType: constants.LineTypeDealRejected, Bucket: constants.BucketRepeatCustomer, TransactionContext: constants.TxVendorSelling, InventoryContext: constants.InvNone, RequestedCount: 10},
		{LineType: constants.LineTypeDealRejected, Bucket: constants.BucketRepeatCustomer, TransactionContext: constants.TxVendorBuying, InventoryContext: constants.InvNone, RequestedCount: 10},

		// Farewells
		{LineType: constants.LineTypeFarewell, Bucket: constants.BucketRepeatCustomer, TransactionContext: constants.TxNeutral, InventoryContext: constants.InvNone, RequestedCount: 10},
	}
}

