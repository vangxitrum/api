package slice

import "github.com/google/uuid"

func DeDupUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]bool)
	result := make([]uuid.UUID, 0, len(ids))

	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result
}
