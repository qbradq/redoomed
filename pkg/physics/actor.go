package physics

const (
	// DefaultPlayerRadius is the standard Doom player collision radius.
	DefaultPlayerRadius = 16.0
	// DefaultPlayerHeight is the standard Doom player physical height.
	DefaultPlayerHeight = 56.0
	// DefaultPlayerEyeHeight is the standard camera eye height above the floor.
	DefaultPlayerEyeHeight = 41.0
	// DefaultMaxStepHeight is the standard Doom maximum step-up height (MAXSTEPMOVE = 24).
	DefaultMaxStepHeight = 24.0
)

// Actor represents a physical entity (player, monster, obstacle) participating in collision and movement.
type Actor struct {
	X             float64
	Y             float64
	Z             float64
	Angle         float64 // Facing angle in degrees (0=East, 90=North, 180=West, 270=South)
	Radius        float64 // 2D collision radius
	Height        float64 // 3D vertical height
	EyeHeight     float64 // Eye offset above floor
	MaxStepHeight float64 // Maximum step up height
	FloorZ        float64 // Current floor height beneath actor
	CeilingZ      float64 // Current ceiling height above actor
	IsMonster     bool    // Whether the actor is a monster (for monster-blocking lines)
}

// NewPlayerActor creates an Actor initialized with standard Doom player physics parameters.
func NewPlayerActor(x, y, z, angle float64) *Actor {
	return &Actor{
		X:             x,
		Y:             y,
		Z:             z,
		Angle:         angle,
		Radius:        DefaultPlayerRadius,
		Height:        DefaultPlayerHeight,
		EyeHeight:     DefaultPlayerEyeHeight,
		MaxStepHeight: DefaultMaxStepHeight,
		FloorZ:        z - DefaultPlayerEyeHeight,
		CeilingZ:      z - DefaultPlayerEyeHeight + DefaultPlayerHeight,
		IsMonster:     false,
	}
}

// NewActor creates a generic physical entity with custom parameters.
func NewActor(x, y, z, angle, radius, height, eyeHeight, maxStepHeight float64, isMonster bool) *Actor {
	return &Actor{
		X:             x,
		Y:             y,
		Z:             z,
		Angle:         angle,
		Radius:        radius,
		Height:        height,
		EyeHeight:     eyeHeight,
		MaxStepHeight: maxStepHeight,
		FloorZ:        z - eyeHeight,
		CeilingZ:      z - eyeHeight + height,
		IsMonster:     isMonster,
	}
}
