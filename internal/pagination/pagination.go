package pagination

const DefaultPage = 1
const DefaultPageSize = 25

type Page[T any] struct {
	Items      []T `json:"items"`
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	TotalItems int `json:"totalItems"`
	TotalPages int `json:"totalPages"`
}

func Normalize(page, pageSize *int32) (int, int) {
	p := DefaultPage
	ps := DefaultPageSize
	if page != nil && *page > 0 {
		p = int(*page)
	}
	if pageSize != nil && *pageSize > 0 {
		ps = int(*pageSize)
	}
	return p, ps
}

func Offset(page, pageSize int) int {
	return (page - 1) * pageSize
}

func TotalPages(totalItems, pageSize int) int {
	if pageSize == 0 {
		return 0
	}
	total := totalItems / pageSize
	if totalItems%pageSize > 0 {
		total++
	}
	return total
}

// Paginate returns the requested page of an in-memory slice (for small catalogues
// that are cheaper to load whole than to page in SQL). Items is never nil.
func Paginate[T any](items []T, page, pageSize *int32) Page[T] {
	p, ps := Normalize(page, pageSize)
	start := min(Offset(p, ps), len(items))
	end := min(start+ps, len(items))
	return Page[T]{
		Items:      append([]T{}, items[start:end]...),
		Page:       p,
		PageSize:   ps,
		TotalItems: len(items),
		TotalPages: TotalPages(len(items), ps),
	}
}
