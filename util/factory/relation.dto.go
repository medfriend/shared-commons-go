package factory

// RelationDTO define los datos para crear una relación FK entre tablas dinámicas.
// - ReferencedTable: tabla objetivo que será referenciada (debe tener columna id BIGSERIAL/ BIGINT)
// - ReferencingTable: tabla que obtendrá la nueva columna y la FK
// - ColumnName: nombre de la columna a crear en la tabla que referencia
// - OnDelete/OnUpdate: acciones de la FK (CASCADE, RESTRICT, SET NULL, NO ACTION); por defecto NO ACTION
// - Nullable: si la columna creada permite NULL
type RelationDTO struct {
	ReferencedTable  string `json:"referenced_table"`
	ReferencingTable string `json:"referencing_table"`
	ColumnName       string `json:"column_name"`
	OnDelete         string `json:"on_delete"`
	OnUpdate         string `json:"on_update"`
	Nullable         bool   `json:"nullable"`
}
