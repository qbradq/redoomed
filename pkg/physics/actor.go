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
	// DefaultGravity is the standard Doom gravitational acceleration per tick (1.0 Doom units/tic^2).
	DefaultGravity = 1.0
	// DefaultMaxFallSpeed is the standard Doom terminal vertical falling velocity (30.0 Doom units/tic).
	DefaultMaxFallSpeed = 30.0
)

// Actor represents a physical entity (player, monster, obstacle) participating in collision and movement.
type Actor struct {
	X             float64
	Y             float64
	Z             float64 // Eye level for player, or elevation in Doom world space
	VelZ          float64 // Vertical velocity in Doom units/tic (+Up, -Down)
	Angle         float64 // Facing angle in degrees (0=East, 90=North, 180=West, 270=South)
	Radius        float64 // 2D collision radius
	Height        float64 // 3D vertical height
	EyeHeight     float64 // Eye offset above floor
	MaxStepHeight float64 // Maximum step up height
	FloorZ        float64 // Current floor height beneath actor
	CeilingZ      float64 // Current ceiling height above actor
	Gravity       float64 // Gravitational acceleration per tick (defaults to DefaultGravity)
	MaxFallSpeed  float64 // Terminal falling velocity (defaults to DefaultMaxFallSpeed)
	IsMonster     bool    // Whether the actor is a monster (for monster-blocking lines)
	NoGravity     bool    // Whether the actor ignores gravity (flying/floating monsters or projectiles)
	NoClip        bool    // Whether the actor ignores wall and sector height collisions (Doom noclip mode)
	OnGround      bool    // True if the actor is supported on the floor
}

// NewPlayerActor creates an Actor initialized with standard Doom player physics parameters.
func NewPlayerActor(x, y, z, angle float64) *Actor {
	return &Actor{
		X:             x,
		Y:             y,
		Z:             z,
		VelZ:          0,
		Angle:         angle,
		Radius:        DefaultPlayerRadius,
		Height:        DefaultPlayerHeight,
		EyeHeight:     DefaultPlayerEyeHeight,
		MaxStepHeight: DefaultMaxStepHeight,
		FloorZ:        z - DefaultPlayerEyeHeight,
		CeilingZ:      z - DefaultPlayerEyeHeight + DefaultPlayerHeight,
		Gravity:       DefaultGravity,
		MaxFallSpeed:  DefaultMaxFallSpeed,
		IsMonster:     false,
		NoGravity:     false,
		OnGround:      true,
	}
}

// NewActor creates a generic physical entity with custom parameters.
func NewActor(x, y, z, angle, radius, height, eyeHeight, maxStepHeight float64, isMonster bool) *Actor {
	return &Actor{
		X:             x,
		Y:             y,
		Z:             z,
		VelZ:          0,
		Angle:         angle,
		Radius:        radius,
		Height:        height,
		EyeHeight:     eyeHeight,
		MaxStepHeight: maxStepHeight,
		FloorZ:        z - eyeHeight,
		CeilingZ:      z - eyeHeight + height,
		Gravity:       DefaultGravity,
		MaxFallSpeed:  DefaultMaxFallSpeed,
		IsMonster:     isMonster,
		NoGravity:     false,
		OnGround:      true,
	}
}
