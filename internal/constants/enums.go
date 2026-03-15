package constants

// LineType values — must match PHP DialogueLineType enum exactly.
const (
	LineTypeGreeting       = "greeting"
	LineTypeInventoryPitch = "inventory_pitch"
	LineTypeDealAccepted   = "deal_accepted"
	LineTypeDealRejected   = "deal_rejected"
	LineTypeFarewell       = "farewell"
)

// InteractionBucket values — must match PHP InteractionBucket enum exactly.
const (
	BucketFirstVisit     = "first_visit"
	BucketSecondVisit    = "second_visit"
	BucketThirdVisit     = "third_visit"
	BucketRepeatCustomer = "repeat_customer"
)

// TransactionContext values — must match PHP TransactionContext enum exactly.
const (
	TxNeutral       = "neutral"
	TxVendorSelling = "vendor_selling"
	TxVendorBuying  = "vendor_buying"
)

// InventoryContext values — must match PHP InventoryContext enum exactly.
const (
	InvNone             = "none"
	InvShip             = "ship"
	InvShieldProjector  = "shield_projector"
	InvEngine           = "engine"
	InvReactor          = "reactor"
	InvWeapon           = "weapon"
	InvSensorArray      = "sensor_array"
	InvCargoModule      = "cargo_module"
	InvHullPlating      = "hull_plating"
	InvSalvageComponent = "salvage_component"
)

// GenerationStatus values — must match PHP dialogue_generation_status enum exactly.
const (
	StatusPending    = "pending"
	StatusGenerating = "generating"
	StatusComplete   = "complete"
	StatusFailed     = "failed"
)
