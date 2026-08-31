package physics

import (
	"math"

	"github.com/qbradq/redoomed/pkg/wad"
)

// CheckItemTouch reports whether an actor is in physical contact with an item entity.
// It checks 2D Euclidean distance against actor and item radii, and verifies vertical overlap.
func CheckItemTouch(actor *Actor, item *wad.ItemEntity) bool {
	if actor == nil || item == nil || item.Collected {
		return false
	}

	dist := math.Hypot(actor.X-item.X, actor.Y-item.Y)
	if dist > (actor.Radius + item.Radius) {
		return false
	}

	// Vertical overlap check:
	// Actor vertical span: [actor.FloorZ, actor.FloorZ + actor.Height]
	// Item vertical span: [item.FloorZ, item.FloorZ + item.Height]
	actorBottom := actor.FloorZ
	actorTop := actor.FloorZ + actor.Height
	itemBottom := item.FloorZ
	itemTop := item.FloorZ + item.Height

	if actorTop < itemBottom || actorBottom > itemTop {
		return false
	}

	return true
}

// TouchItems tests the actor against a slice of item entities and returns any that are touched.
func TouchItems(actor *Actor, items []*wad.ItemEntity) []*wad.ItemEntity {
	if actor == nil || len(items) == 0 {
		return nil
	}

	var touched []*wad.ItemEntity
	for _, item := range items {
		if CheckItemTouch(actor, item) {
			touched = append(touched, item)
		}
	}
	return touched
}
