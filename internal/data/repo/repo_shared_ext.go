package repo

import (
	"fmt"
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

func resolvePaging(req *paginationv1.PagingRequest) (offset int, limit int, apply bool, err error) {
	if req == nil || req.GetNoPaging() {
		return 0, 0, false, nil
	}

	if req.GetLimit() > 0 {
		return int(req.GetOffset()), int(req.GetLimit()), true, nil
	}

	if req.GetPageSize() > 0 {
		page := req.GetPage()
		if page == 0 {
			page = 1
		}
		pageSize := req.GetPageSize()
		return int((page - 1) * pageSize), int(pageSize), true, nil
	}

	if req.GetOffset() > 0 {
		return int(req.GetOffset()), 20, true, nil
	}

	return 0, 20, true, nil
}

func firstSorting(req *paginationv1.PagingRequest) (string, paginationv1.Sorting_Direction, error) {
	if req == nil || len(req.GetSorting()) == 0 {
		return "", paginationv1.Sorting_ASC, nil
	}

	item := req.GetSorting()[0]
	if item == nil {
		return "", paginationv1.Sorting_ASC, nil
	}

	field := strings.TrimSpace(item.GetField())
	if field == "" {
		return "", paginationv1.Sorting_ASC, fmt.Errorf("sorting field is empty")
	}

	direction := item.GetDirection()
	if direction == 0 {
		direction = paginationv1.Sorting_ASC
	}

	return field, direction, nil
}
