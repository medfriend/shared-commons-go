package factory

// InsertDTO representa la inserción de una fila en una tabla dinámica
// Data admite claves válidas según el esquema: id (no usar en insert), tipo, variable,
// variable_type, value, is_public, public_token, user_created. date_created es por defecto.
type InsertDTO struct {
	Data map[string]interface{} `json:"data"`
}

// DeleteDTO representa un borrado por filtros (se requiere al menos un filtro)
type DeleteDTO struct {
	Filters []FilterCondition `json:"filters"`
}
