package ai

// BulletInfo represents minimal bullet information needed by AI
// This avoids import cycle with game package
type BulletInfo struct {
	X     int
	Y     int
	DirX  int
	DirY  int
	Alive bool
}

// DetectBulletThreat checks for bullets within 7 cells that are moving toward the enemy
// Returns [4]bool indicating threat from: [up, down, left, right]
// A threat is detected when a bullet is:
// 1. Within 7 cells in a cardinal direction
// 2. Moving toward the enemy (not away)
func DetectBulletThreat(enemyX, enemyY int, bullets []BulletInfo) [4]bool {
	var threats [4]bool
	const maxRange = 7

	for _, bullet := range bullets {
		if !bullet.Alive {
			continue
		}

		// Calculate distance and direction from enemy to bullet
		dx := bullet.X - enemyX
		dy := bullet.Y - enemyY

		// Check if bullet is within range
		if abs(dx) > maxRange || abs(dy) > maxRange {
			continue
		}

		// Determine which direction the threat is coming from
		// and verify bullet is moving toward enemy
		if dx == 0 && dy != 0 {
			// Bullet is directly above or below
			if dy < 0 && bullet.DirY > 0 {
				// Bullet is above and moving down (toward enemy)
				threats[0] = true // Up
			} else if dy > 0 && bullet.DirY < 0 {
				// Bullet is below and moving up (toward enemy)
				threats[1] = true // Down
			}
		} else if dy == 0 && dx != 0 {
			// Bullet is directly left or right
			if dx < 0 && bullet.DirX > 0 {
				// Bullet is to the left and moving right (toward enemy)
				threats[2] = true // Left
			} else if dx > 0 && bullet.DirX < 0 {
				// Bullet is to the right and moving left (toward enemy)
				threats[3] = true // Right
			}
		}
		// Bullets at diagonal positions are ignored to keep state space manageable
	}

	return threats
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
