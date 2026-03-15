package vendors

import (
	"encoding/json"
	"time"
)

type Personality map[string]float64

type VendorProfile struct {
	ID                        int64
	UUID                      string
	ServiceType               string
	Criminality               float64
	MarkupBase                float64
	PersonalityJSON           string
	Personality               Personality
	DialogueGenerationStatus  string
	DialogueGenerationVersion int
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

func (vp *VendorProfile) ParsePersonality() error {
	if vp.PersonalityJSON == "" {
		vp.Personality = make(Personality)
		return nil
	}
	return json.Unmarshal([]byte(vp.PersonalityJSON), &vp.Personality)
}

// Get returns a personality trait value with a default of 0.5 if not found.
func (p Personality) Get(key string) float64 {
	if p == nil {
		return 0.5
	}
	if val, exists := p[key]; exists {
		return val
	}
	return 0.5
}
