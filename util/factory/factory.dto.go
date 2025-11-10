package factory

// CreateTableDTO define los datos para crear una nueva tabla dinámica tipo "patrón".
type CreateTableDTO struct {
	// Nombre de la tabla a crear (solo letras, números y guiones bajos)
	Name string `json:"name"`
	// OPCIONAL (modo legacy de patrón): Tipo de dato principal del campo "value" (integer, decimal, text, boolean, datetime, json)
	Tipo string `json:"tipo"`
	// OPCIONAL (legacy): Tipo de la variable que se medirá (informativo)
	VariableType string `json:"variable_type"`
	// OPCIONAL (legacy): Indica si los datos serán públicos
	IsPublic bool `json:"is_public"`
	// OPCIONAL (legacy): Usuario creador (id)
	UserCreated uint64 `json:"user_created"`
	// NUEVO: Definición general de columnas. Si se provee, se ignora el modo legacy anterior
	Columns []ColumnDef `json:"columns"`
}

// ColumnDef define una columna genérica para CreateTable
type ColumnDef struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"` // tipo SQL (p.ej. "bigserial", "bigint", "varchar(200)", "jsonb", "timestamptz")
	Nullable   bool    `json:"nullable"`
	Default    *string `json:"default"` // literal seguro (número, boolean, o cadena entre comillas). Expresiones avanzadas no soportadas.
	PrimaryKey bool    `json:"primary_key"`
	Unique     bool    `json:"unique"`
	References *RefDef `json:"references"` // FK opcional
}

// RefDef describe una FK simple (columna única)
type RefDef struct {
	Table    string `json:"table"`
	Column   string `json:"column"`
	OnDelete string `json:"on_delete"`
	OnUpdate string `json:"on_update"`
}

// FilterCondition representa un filtro estilo Directus
// op soportados: eq, neq, gt, gte, lt, lte, like, ilike, in, nin, null, nnull, between, contains
type FilterCondition struct {
	Field string      `json:"field"`
	Op    string      `json:"op"`
	Value interface{} `json:"value"`
}

// QueryDTO para lectura con libertad de filtros, selección, orden y paginación
type QueryDTO struct {
	Filters []FilterCondition `json:"filters"`
	Fields  []string          `json:"fields"`
	Sort    []string          `json:"sort"` // ejemplo: ["-date_created", "variable"]
	Limit   int               `json:"limit"`
	Offset  int               `json:"offset"`
	// Logic define cómo combinar los filtros a nivel superior: "AND" (por defecto) o "OR"
	Logic string `json:"logic"`
	// Joins permite unir tablas adicionales. Los campos y filtros pueden referenciar alias: "alias.campo"
	Joins []JoinDTO `json:"joins"`
	// Pagined activa paginación por página y tamaño. Si se provee, sobreescribe limit/offset.
	Pagined *PaginedDTO `json:"pagined"`
}

// JoinDTO describe una unión con otra tabla
type JoinDTO struct {
	// Type: "LEFT" | "INNER" | "RIGHT" (case-insensitive). Por defecto: LEFT
	Type string `json:"type"`
	// Table: nombre de la tabla a unir
	Table string `json:"table"`
	// Alias: alias para la tabla unida; requerido si hay múltiples joins
	Alias string `json:"alias"`
	// On: una o más condiciones de unión (se combinan con AND)
	On []JoinOnDTO `json:"on"`
}

// JoinOnDTO representa una condición de unión: left = right.
// Cada lado debe ser un identificador calificado: "alias.columna"
type JoinOnDTO struct {
	Left  string `json:"left"`
	Right string `json:"right"`
}

// PaginedDTO define paginación por página
type PaginedDTO struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

// UpdateDTO para actualizar con filtros
type UpdateDTO struct {
	Data    map[string]interface{} `json:"data"`
	Filters []FilterCondition      `json:"filters"`
}
