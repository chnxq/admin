package repo

import (
	"strings"

	paginationv1 "github.com/chnxq/x-crud/api/gen/pagination/v1"
)

func withDefaultSorting(
	items []*paginationv1.Sorting,
	field string,
	direction paginationv1.Sorting_Direction,
) []*paginationv1.Sorting {
	if len(items) > 0 {
		return items
	}
	field = strings.TrimSpace(field)
	if field == "" {
		return items
	}
	return []*paginationv1.Sorting{
		{
			Field:     field,
			Direction: direction,
		},
	}
}
