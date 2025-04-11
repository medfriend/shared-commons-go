package dto

type PaginationDTO struct {
	Pagina uint `json:"pagina"`
	Filas  uint `json:"filas"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalPages int         `json:"totalPages"`
}
