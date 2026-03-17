package prompts

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"space-wars-3002-text-generation/internal/vendors"
)

// Prompt holds a system message, user message, and a hash for logging/diagnostics.
type Prompt struct {
	System string
	User   string
	Hash   string
}

// BuildPrompt constructs the LLM prompt for a specific dialogue scope.
// lineType:           greeting|inventory_pitch|deal_accepted|deal_rejected|farewell
// bucket:             first_visit|second_visit|third_visit|repeat_customer
// transactionContext: neutral|vendor_selling|vendor_buying
// inventoryContext:   none|ship|engine|etc.
// count:              number of lines to request from the LLM
func BuildPrompt(vendor vendors.VendorProfile, lineType, bucket, transactionContext, inventoryContext string, count int) *Prompt {
	system := buildSystemPrompt()
	user := buildUserPrompt(vendor, lineType, bucket, transactionContext, inventoryContext, count)

	normalized := strings.TrimSpace(system + "\n" + user)
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(normalized)))

	return &Prompt{
		System: system,
		User:   user,
		Hash:   hash,
	}
}

// buildSystemPrompt returns the invariant system message per design §9.1.
func buildSystemPrompt() string {
	return "You generate short in-universe dialogue for NPC vendors in a space exploration and trading game.\n" +
		"Each line must be one sentence.\n" +
		"Do not include explanations, preamble, or commentary.\n" +
		"Do not include markdown."
}

// buildUserPrompt assembles the vendor-specific user message per design §9.2.
func buildUserPrompt(vendor vendors.VendorProfile, lineType, bucket, transactionContext, inventoryContext string, count int) string {
	var b strings.Builder

	serviceTypeTone := deriveServiceTypeTone(vendor.ServiceType)
	criminalityTone := deriveCriminalityTone(vendor.Criminality)
	markupTone := deriveMarkupTone(vendor.MarkupBase)
	honestyHint := deriveHonestyHint(vendor.Personality.Get("honesty"))
	greedHint := deriveGreedHint(vendor.Personality.Get("greed"))
	charmHint := deriveCharmHint(vendor.Personality.Get("charm"))
	riskHint := deriveRiskHint(vendor.Personality.Get("risk_tolerance"))

	minWords, maxWords := lineTypeWordLimits(lineType)

	fmt.Fprintf(&b, "Generate %d %s lines for a %s vendor.\n\n", count, lineType, vendor.ServiceType)

	b.WriteString("Context:\n")
	fmt.Fprintf(&b, "- service_type: %s (%s)\n", vendor.ServiceType, serviceTypeTone)
	fmt.Fprintf(&b, "- interaction_bucket: %s\n", bucket)
	fmt.Fprintf(&b, "- transaction_context: %s\n", transactionContext)
	fmt.Fprintf(&b, "- inventory_context: %s\n", inventoryContext)
	fmt.Fprintf(&b, "- criminality_tone: %s\n", criminalityTone)
	fmt.Fprintf(&b, "- markup_tone: %s\n\n", markupTone)

	b.WriteString("Vendor voice:\n")
	fmt.Fprintf(&b, "- honesty_hint: %s\n", honestyHint)
	fmt.Fprintf(&b, "- greed_hint: %s\n", greedHint)
	fmt.Fprintf(&b, "- charm_hint: %s\n", charmHint)
	fmt.Fprintf(&b, "- risk_hint: %s\n\n", riskHint)

	b.WriteString("Rules:\n")
	b.WriteString("- one sentence per line\n")
	fmt.Fprintf(&b, "- %d to %d words per line\n", minWords, maxWords)
	b.WriteString("- keep lines in character\n")
	b.WriteString("- do not mention exact live item condition percentages\n")
	b.WriteString("- do not invent exact live defects\n")
	b.WriteString("- number each line: 1. 2. 3. etc.\n")
	b.WriteString("- output the numbered list only, nothing else\n")

	return b.String()
}

// deriveServiceTypeTone maps service_type to a human-readable tone hint per design §8.1.
func deriveServiceTypeTone(serviceType string) string {
	switch serviceType {
	case "salvage_yard":
		return "used goods dealer, practical tone, rough edges allowed, may reference salvage and wear"
	case "shipyard":
		return "polished and professional, prideful about product quality, stronger sales posture"
	case "trading_hub":
		return "general merchant, broad product framing, commercial tone"
	case "market":
		return "open commerce marketplace, general trade language, less technical specificity"
	default:
		return "general merchant in a space trading hub"
	}
}

// deriveCriminalityTone maps 0.0–1.0 criminality to a tone hint per design §8.2.
func deriveCriminalityTone(criminality float64) string {
	switch {
	case criminality >= 0.7:
		return "predatory edge, rougher language, dubious but within content rules"
	case criminality >= 0.4:
		return "sharper tone, more opportunistic, some roughness permitted"
	default:
		return "cleaner language, low menace, straightforward"
	}
}

// deriveMarkupTone maps markup_base to a price posture hint per design §8.3.
func deriveMarkupTone(markupBase float64) string {
	switch {
	case markupBase > 0.3:
		return "defensive about pricing, premium framing, worth-the-price language"
	case markupBase < 0.05:
		return "casual bargain framing, lower price defensiveness, less pompous"
	default:
		return "moderate pricing stance, matter-of-fact"
	}
}

// deriveHonestyHint maps the honesty trait to a voice hint per design §8.4.
func deriveHonestyHint(honesty float64) string {
	switch {
	case honesty >= 0.7:
		return "blunt, transparent, straightforward, minimal exaggeration"
	case honesty <= 0.3:
		return "evasive, exaggerates benefits, downplays flaws"
	default:
		return "occasionally candid but still a salesperson"
	}
}

// deriveGreedHint maps the greed trait to a price defensiveness hint per design §8.4.
func deriveGreedHint(greed float64) string {
	switch {
	case greed >= 0.7:
		return "hard sell, price defensive, scarcity framing, pushy"
	case greed <= 0.3:
		return "relaxed about price, not pushing hard for margin"
	default:
		return "wants to close the deal, mild price defensiveness"
	}
}

// deriveCharmHint maps the charm trait to a social tone hint per design §8.4.
func deriveCharmHint(charm float64) string {
	switch {
	case charm >= 0.7:
		return "smooth, friendly, socially at ease, persuasive without pressure"
	case charm <= 0.3:
		return "blunt, socially curt, no pleasantries"
	default:
		return "professional but not especially warm"
	}
}

// deriveRiskHint maps risk_tolerance to a willingness to hype questionable goods per design §8.4.
func deriveRiskHint(riskTolerance float64) string {
	switch {
	case riskTolerance >= 0.7:
		return "happy to sell questionable goods, rough confidence, used-but-usable framing"
	case riskTolerance <= 0.3:
		return "only sells clearly legitimate goods, conservative framing"
	default:
		return "will sell grey-area goods but frames them diplomatically"
	}
}

// lineTypeWordLimits returns the min and max word counts for a given line type.
func lineTypeWordLimits(lineType string) (min, max int) {
	switch lineType {
	case "greeting":
		return 6, 16
	case "inventory_pitch":
		return 8, 18
	case "deal_accepted", "deal_rejected":
		return 6, 14
	case "farewell":
		return 6, 14
	default:
		return 6, 20
	}
}
