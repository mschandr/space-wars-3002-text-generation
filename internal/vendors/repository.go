package vendors

import (
	"fmt"
	"time"

	"space-wars-3002-text-generation/internal/db"
	"space-wars-3002-text-generation/internal/logging"
)

// FetchPending loads vendors with dialogue_generation_status IN ('pending', 'failed').
// Queries galaxy_vendor_profiles joined to vendor_profiles for personality/markup data.
func FetchPending(database *db.DB, limit int, logger *logging.Logger) ([]VendorProfile, error) {
	rows, err := database.Query(`
		SELECT
			gvp.id,
			gvp.uuid,
			gvp.service_type,
			gvp.criminality,
			vp.markup_base,
			vp.personality,
			gvp.dialogue_generation_status,
			gvp.dialogue_generation_version,
			gvp.created_at,
			gvp.updated_at
		FROM galaxy_vendor_profiles gvp
		JOIN vendor_profiles vp ON vp.id = gvp.vendor_profile_id
		WHERE gvp.dialogue_generation_status IN ('pending', 'failed')
		ORDER BY gvp.id
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pending vendors: %w", err)
	}
	defer rows.Close()

	var vendors []VendorProfile
	for rows.Next() {
		var v VendorProfile
		err := rows.Scan(
			&v.ID,
			&v.UUID,
			&v.ServiceType,
			&v.Criminality,
			&v.MarkupBase,
			&v.PersonalityJSON,
			&v.DialogueGenerationStatus,
			&v.DialogueGenerationVersion,
			&v.CreatedAt,
			&v.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan vendor: %w", err)
		}

		if err := v.ParsePersonality(); err != nil {
			logger.Warnf("skipping vendor with malformed personality JSON", map[string]interface{}{
				"vendor_id": v.ID,
				"error":     err.Error(),
			})
			continue
		}

		vendors = append(vendors, v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating vendors: %w", err)
	}

	return vendors, nil
}

// MarkGenerating sets dialogue_generation_status = 'generating' for a galaxy vendor profile.
func MarkGenerating(database *db.DB, vendorID int64) error {
	_, err := database.Exec(`
		UPDATE galaxy_vendor_profiles
		SET dialogue_generation_status = 'generating'
		WHERE id = ?
	`, vendorID)
	if err != nil {
		return fmt.Errorf("failed to mark vendor %d as generating: %w", vendorID, err)
	}
	return nil
}

// MarkComplete sets status = 'complete' and records the generation timestamp.
func MarkComplete(database *db.DB, vendorID int64) error {
	_, err := database.Exec(`
		UPDATE galaxy_vendor_profiles
		SET dialogue_generation_status = 'complete',
		    dialogue_generated_at = ?
		WHERE id = ?
	`, time.Now().UTC().Format("2006-01-02 15:04:05"), vendorID)
	if err != nil {
		return fmt.Errorf("failed to mark vendor %d as complete: %w", vendorID, err)
	}
	return nil
}

// MarkFailed sets dialogue_generation_status = 'failed'.
func MarkFailed(database *db.DB, vendorID int64) error {
	_, err := database.Exec(`
		UPDATE galaxy_vendor_profiles
		SET dialogue_generation_status = 'failed'
		WHERE id = ?
	`, vendorID)
	if err != nil {
		return fmt.Errorf("failed to mark vendor %d as failed: %w", vendorID, err)
	}
	return nil
}
