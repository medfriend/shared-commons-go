package factory

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strings"
	"time"

	scdto "github.com/medfriend/shared-commons-go/dto"
	baseRepo "github.com/medfriend/shared-commons-go/util/repository"
	"gorm.io/gorm"
)

// FactoryRepository define operaciones CRUD/DDL dinámicas estilo Directus.
type FactoryRepository interface {
	CreateTable(input CreateTableDTO) error
	Query(table string, q QueryDTO) (scdto.PaginatedResponse, error)
	Update(table string, upd UpdateDTO) (int64, error)
	DropTable(table string) error
	CreateRelation(rel RelationDTO) error
	Insert(table string, data map[string]interface{}) (uint64, error)
	Delete(table string, q DeleteDTO) (int64, error)
}

type FactoryRepositoryImpl struct {
	DB   *gorm.DB
	Base baseRepo.BaseRepository[map[string]interface{}]
}

var (
	// Tabla/columna seguras: letras, números y guión bajo.
	safeIdentRe = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

var allowedTipos = map[string]struct{}{
	"integer":  {},
	"decimal":  {},
	"text":     {},
	"boolean":  {},
	"datetime": {},
	"json":     {},
}

func normalizeTipo(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func validateValueAgainstTipo(tipo string, val interface{}) error {
	t := normalizeTipo(tipo)
	if _, ok := allowedTipos[t]; !ok {
		return fmt.Errorf("tipo inválido: %s", tipo)
	}
	if val == nil {
		return nil
	}
	switch t {
	case "integer":
		switch v := val.(type) {
		case float64:
			if math.Mod(v, 1) != 0 {
				return errors.New("value debe ser entero para tipo=integer")
			}
		case int, int32, int64, uint, uint32, uint64:
			// ok
		default:
			return errors.New("value debe ser entero para tipo=integer")
		}
	case "decimal":
		switch val.(type) {
		case float64, int, int32, int64, uint, uint32, uint64:
			// ok
		default:
			return errors.New("value debe ser numérico para tipo=decimal")
		}
	case "text":
		if _, ok := val.(string); !ok {
			return errors.New("value debe ser string para tipo=text")
		}
	case "boolean":
		if _, ok := val.(bool); !ok {
			return errors.New("value debe ser boolean para tipo=boolean")
		}
	case "datetime":
		s, ok := val.(string)
		if !ok {
			return errors.New("value debe ser string RFC3339 para tipo=datetime")
		}
		// Aceptar varias formas comunes
		layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"}
		var parsed bool
		for _, l := range layouts {
			if _, err := time.Parse(l, s); err == nil {
				parsed = true
				break
			}
		}
		if !parsed {
			return errors.New("value datetime inválido; usa RFC3339 o 'YYYY-MM-DD HH:MM:SS'")
		}
	case "json":
		switch val.(type) {
		case map[string]interface{}, []interface{}:
			// ok
		default:
			return errors.New("value debe ser objeto o arreglo para tipo=json")
		}
	}
	return nil
}

func NewFactoryRepository(db *gorm.DB) FactoryRepository {
	return &FactoryRepositoryImpl{DB: db, Base: baseRepo.BaseRepository[map[string]interface{}]{DB: db}}
}

func (r *FactoryRepositoryImpl) validateIdent(name string) error {
	if !safeIdentRe.MatchString(name) {
		return fmt.Errorf("identificador inválido: %s", name)
	}
	return nil
}

func (r *FactoryRepositoryImpl) CreateTable(input CreateTableDTO) error {
	if err := r.validateIdent(input.Name); err != nil {
		return err
	}
	if len(input.Columns) == 0 {
		return errors.New("se requiere 'columns' para definir la tabla")
	}
	return r.createTableGeneric(input)
}

// createTableGeneric crea una tabla con definición arbitraria de columnas y FKs opcionales
func (r *FactoryRepositoryImpl) createTableGeneric(input CreateTableDTO) error {
	if len(input.Columns) == 0 {
		return errors.New("se requiere al menos una columna")
	}

	colDefs := make([]string, 0, len(input.Columns))
	pkCols := make([]string, 0)
	for _, c := range input.Columns {
		if err := r.validateIdent(c.Name); err != nil {
			return err
		}
		sqlType, err := validateSQLType(c.Type)
		if err != nil {
			return err
		}
		nullSQL := "NULL"
		if !c.Nullable {
			nullSQL = "NOT NULL"
		}
		def := fmt.Sprintf("\"%s\" %s %s", c.Name, sqlType, nullSQL)
		if c.Default != nil {
			if !isSafeDefaultLiteral(*c.Default) {
				return fmt.Errorf("default inválido para columna %s", c.Name)
			}
			def += fmt.Sprintf(" DEFAULT %s", *c.Default)
		}
		if c.Unique {
			def += " UNIQUE"
		}
		colDefs = append(colDefs, def)
		if c.PrimaryKey {
			pkCols = append(pkCols, fmt.Sprintf("\"%s\"", c.Name))
		}
	}
	if len(pkCols) > 0 {
		colDefs = append(colDefs, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pkCols, ",")))
	}

	createSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS \"%s\" (%s);", input.Name, strings.Join(colDefs, ", "))

	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(createSQL).Error; err != nil {
			return err
		}
		// Agregar FKs idempotentes
		for _, c := range input.Columns {
			if c.References == nil {
				continue
			}
			ref := c.References
			if err := r.validateIdent(ref.Table); err != nil {
				return err
			}
			if err := r.validateIdent(ref.Column); err != nil {
				return err
			}
			onDelete := strings.ToUpper(strings.TrimSpace(ref.OnDelete))
			onUpdate := strings.ToUpper(strings.TrimSpace(ref.OnUpdate))
			if onDelete == "" {
				onDelete = "NO ACTION"
			}
			if onUpdate == "" {
				onUpdate = "NO ACTION"
			}
			validActions := map[string]struct{}{"NO ACTION": {}, "CASCADE": {}, "RESTRICT": {}, "SET NULL": {}}
			if _, ok := validActions[onDelete]; !ok {
				return fmt.Errorf("on_delete inválido: %s", onDelete)
			}
			if _, ok := validActions[onUpdate]; !ok {
				return fmt.Errorf("on_update inválido: %s", onUpdate)
			}
			constraint := fmt.Sprintf("fk_%s_%s_%s", input.Name, ref.Table, c.Name)
			addFk := fmt.Sprintf(`DO $$
        BEGIN
            IF NOT EXISTS (
                SELECT 1 FROM information_schema.table_constraints tc
                WHERE tc.constraint_name = '%s'
                  AND tc.table_name = '%s'
                  AND tc.constraint_type = 'FOREIGN KEY'
            ) THEN
                ALTER TABLE "%s"
                ADD CONSTRAINT "%s" FOREIGN KEY ("%s")
                REFERENCES "%s"("%s") ON DELETE %s ON UPDATE %s;
            END IF;
        END$$;`, constraint, input.Name, input.Name, constraint, c.Name, ref.Table, ref.Column, onDelete, onUpdate)
			if err := tx.Exec(addFk).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// buildQuery aplica filtros, selección, orden, límite y offset al *gorm.DB
func (r *FactoryRepositoryImpl) buildQuery(qb *gorm.DB, q QueryDTO, table string) (*gorm.DB, error) {
	// Asignar alias base para la tabla principal
	baseAlias := "t0"
	qb = qb.Table(fmt.Sprintf("\"%s\" AS \"%s\"", table, baseAlias))

	// Procesar joins si existen
	for _, j := range q.Joins {
		joinType := strings.ToUpper(strings.TrimSpace(j.Type))
		if joinType == "" {
			joinType = "LEFT"
		}
		if joinType != "LEFT" && joinType != "INNER" && joinType != "RIGHT" {
			return nil, fmt.Errorf("tipo de join inválido: %s", j.Type)
		}
		if err := r.validateIdent(j.Table); err != nil {
			return nil, err
		}
		if j.Alias == "" {
			return nil, fmt.Errorf("alias requerido en join para tabla %s", j.Table)
		}
		if err := r.validateIdent(j.Alias); err != nil {
			return nil, err
		}
		if len(j.On) == 0 {
			return nil, fmt.Errorf("join %s requiere al menos una condición ON", j.Alias)
		}

		onClauses := make([]string, 0, len(j.On))
		for _, on := range j.On {
			if err := r.validateQualified(on.Left); err != nil {
				return nil, err
			}
			if err := r.validateQualified(on.Right); err != nil {
				return nil, err
			}
			onClauses = append(onClauses, fmt.Sprintf("%s = %s", quoteQualified(on.Left), quoteQualified(on.Right)))
		}
		joinSQL := fmt.Sprintf("%s JOIN \"%s\" AS \"%s\" ON %s", joinType, j.Table, j.Alias, strings.Join(onClauses, " AND "))
		qb = qb.Joins(joinSQL)
	}
	// Selección de campos
	if len(q.Fields) > 0 {
		// Validar identificadores y seleccionar dinámicamente (acepta alias.campo)
		fields := make([]string, 0, len(q.Fields))
		for _, f := range q.Fields {
			if strings.Contains(f, ".") {
				if err := r.validateQualified(f); err != nil {
					return nil, err
				}
				fields = append(fields, quoteQualified(f))
			} else {
				if err := r.validateIdent(f); err != nil {
					return nil, err
				}
				fields = append(fields, quoteQualified(baseAlias+"."+f))
			}
		}
		qb = qb.Select(strings.Join(fields, ", "))
	}

	// Filtros (AND por defecto; OR si q.Logic == "OR")
	logic := strings.ToUpper(strings.TrimSpace(q.Logic))
	if logic == "OR" && len(q.Filters) > 0 {
		type clause struct {
			sql  string
			args []interface{}
		}
		clauses := make([]clause, 0, len(q.Filters))
		for _, f := range q.Filters {
			if err := r.validateQualified(f.Field); err != nil {
				return nil, err
			}
			sql, args, err := r.filterToClause(f, baseAlias)
			if err != nil {
				return nil, err
			}
			clauses = append(clauses, clause{sql: sql, args: args})
		}
		qb = qb.Where(func(tx *gorm.DB) *gorm.DB {
			first := true
			for _, c := range clauses {
				if first {
					tx = tx.Where(c.sql, c.args...)
					first = false
				} else {
					tx = tx.Or(c.sql, c.args...)
				}
			}
			return tx
		})
	} else {
		for _, f := range q.Filters {
			if err := r.validateQualified(f.Field); err != nil {
				return nil, err
			}
			sql, args, err := r.filterToClause(f, baseAlias)
			if err != nil {
				return nil, err
			}
			qb = qb.Where(sql, args...)
		}
	}

	// Orden
	for _, s := range q.Sort {
		dir := "ASC"
		field := s
		if strings.HasPrefix(s, "-") {
			dir = "DESC"
			field = strings.TrimPrefix(s, "-")
		}
		if err := r.validateQualified(field); err != nil {
			return nil, err
		}
		// permitir cualquier campo válido como identificador (alias.campo)
		// Si no viene con alias, asumir base alias
		orderExpr := ""
		if strings.Contains(field, ".") {
			orderExpr = fmt.Sprintf("%s %s", quoteQualified(field), dir)
		} else {
			orderExpr = fmt.Sprintf("%s %s", quoteQualified(baseAlias+"."+field), dir)
		}
		qb = qb.Order(orderExpr)
	}

	// Paginación
	if q.Limit > 0 {
		qb = qb.Limit(q.Limit)
	}
	if q.Offset > 0 {
		qb = qb.Offset(q.Offset)
	}

	return qb, nil
}

// filterToClause convierte un filtro en un fragmento SQL y sus argumentos
func (r *FactoryRepositoryImpl) filterToClause(f FilterCondition, baseAlias string) (string, []interface{}, error) {
	op := strings.ToLower(f.Op)
	// Determinar alias y columna
	alias, col := parseAliasAndCol(f.Field, baseAlias)
	if col == "value" {
		valueText := fmt.Sprintf("trim('\"' from \"%s\".\"value\"::text)", alias)
		numericExpr := fmt.Sprintf("(CASE WHEN \"%s\".\"tipo\" IN ('integer','decimal') THEN (%s)::numeric END)", alias, valueText)
		datetimeExpr := fmt.Sprintf("(CASE WHEN \"%s\".\"tipo\" = 'datetime' THEN (%s)::timestamptz END)", alias, valueText)
		switch op {
		case "null":
			return fmt.Sprintf("\"%s\".\"value\" IS NULL", alias), nil, nil
		case "nnull":
			return fmt.Sprintf("\"%s\".\"value\" IS NOT NULL", alias), nil, nil
		case "eq":
			return valueText + " = ?", []interface{}{toString(f.Value)}, nil
		case "neq":
			return valueText + " <> ?", []interface{}{toString(f.Value)}, nil
		case "like":
			return valueText + " LIKE ?", []interface{}{toLike(toString(f.Value))}, nil
		case "ilike":
			return valueText + " ILIKE ?", []interface{}{toLike(toString(f.Value))}, nil
		case "contains":
			return valueText + " LIKE ?", []interface{}{toLike(toString(f.Value))}, nil
		case "in":
			return valueText + " IN (?)", []interface{}{toStringSlice(f.Value)}, nil
		case "nin":
			return valueText + " NOT IN (?)", []interface{}{toStringSlice(f.Value)}, nil
		case "gt", "gte", "lt", "lte":
			cmp := map[string]string{"gt": ">", "gte": ">=", "lt": "<", "lte": "<="}[op]
			switch v := f.Value.(type) {
			case float64, int, int64, uint64:
				return numericExpr + " " + cmp + " ?", []interface{}{v}, nil
			default:
				return datetimeExpr + " " + cmp + " ?::timestamptz", []interface{}{toString(f.Value)}, nil
			}
		case "between":
			vals, ok := f.Value.([]interface{})
			if !ok || len(vals) != 2 {
				return "", nil, errors.New("between requiere array de 2 valores")
			}
			switch vals[0].(type) {
			case float64, int, int64, uint64:
				return numericExpr + " BETWEEN ? AND ?", []interface{}{vals[0], vals[1]}, nil
			default:
				return datetimeExpr + " BETWEEN ?::timestamptz AND ?::timestamptz", []interface{}{toString(vals[0]), toString(vals[1])}, nil
			}
		default:
			return "", nil, fmt.Errorf("operador no soportado para value: %s", f.Op)
		}
	}

	// Columna general (posible alias.campo)
	qcol := quoteRef(f.Field, baseAlias)
	switch op {
	case "eq":
		return fmt.Sprintf("%s = ?", qcol), []interface{}{f.Value}, nil
	case "neq":
		return fmt.Sprintf("%s <> ?", qcol), []interface{}{f.Value}, nil
	case "gt":
		return fmt.Sprintf("%s > ?", qcol), []interface{}{f.Value}, nil
	case "gte":
		return fmt.Sprintf("%s >= ?", qcol), []interface{}{f.Value}, nil
	case "lt":
		return fmt.Sprintf("%s < ?", qcol), []interface{}{f.Value}, nil
	case "lte":
		return fmt.Sprintf("%s <= ?", qcol), []interface{}{f.Value}, nil
	case "like":
		return fmt.Sprintf("%s LIKE ?", qcol), []interface{}{f.Value}, nil
	case "ilike":
		return fmt.Sprintf("%s ILIKE ?", qcol), []interface{}{f.Value}, nil
	case "in":
		return fmt.Sprintf("%s IN (?)", qcol), []interface{}{f.Value}, nil
	case "nin":
		return fmt.Sprintf("%s NOT IN (?)", qcol), []interface{}{f.Value}, nil
	case "null":
		return fmt.Sprintf("%s IS NULL", qcol), nil, nil
	case "nnull":
		return fmt.Sprintf("%s IS NOT NULL", qcol), nil, nil
	case "between":
		vals, ok := f.Value.([]interface{})
		if !ok || len(vals) != 2 {
			return "", nil, errors.New("between requiere array de 2 valores")
		}
		return fmt.Sprintf("%s BETWEEN ? AND ?", qcol), []interface{}{vals[0], vals[1]}, nil
	case "contains":
		return fmt.Sprintf("%s LIKE ?", qcol), []interface{}{toLike(toString(f.Value))}, nil
	default:
		return "", nil, fmt.Errorf("operador no soportado: %s", f.Op)
	}
}

// validateQualified valida alias.campo o campo simple
func (r *FactoryRepositoryImpl) validateQualified(name string) error {
	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		if len(parts) != 2 {
			return fmt.Errorf("identificador calificado inválido: %s", name)
		}
		if err := r.validateIdent(parts[0]); err != nil {
			return err
		}
		if err := r.validateIdent(parts[1]); err != nil {
			return err
		}
		return nil
	}
	return r.validateIdent(name)
}

// quoteQualified convierte alias.campo o campo en referencia con comillas
func quoteQualified(name string) string {
	if strings.Contains(name, ".") {
		parts := strings.SplitN(name, ".", 2)
		return fmt.Sprintf("\"%s\".\"%s\"", parts[0], parts[1])
	}
	return fmt.Sprintf("\"%s\"", name)
}

// parseAliasAndCol devuelve alias y columna, usando baseAlias si no se provee
func parseAliasAndCol(name string, baseAlias string) (string, string) {
	if strings.Contains(name, ".") {
		parts := strings.SplitN(name, ".", 2)
		return parts[0], parts[1]
	}
	return baseAlias, name
}

// quoteRef cita campo, asumiendo alias base si no se provee
func quoteRef(name string, baseAlias string) string {
	if strings.Contains(name, ".") {
		return quoteQualified(name)
	}
	return quoteQualified(baseAlias + "." + name)
}

func (r *FactoryRepositoryImpl) Query(table string, q QueryDTO) (scdto.PaginatedResponse, error) {
	if err := r.validateIdent(table); err != nil {
		return scdto.PaginatedResponse{}, err
	}

	qb := r.DB.Table(table)
	qb, err := r.buildQuery(qb, q, table)
	if err != nil {
		return scdto.PaginatedResponse{}, err
	}

	// Paginación por páginas (opcional)
	var (
		page       int   = 1
		pageSize   int   = 0
		total      int64 = 0
		totalPages int   = 0
	)
	if q.Pagined != nil && q.Pagined.PageSize > 0 {
		if q.Pagined.Page > 0 {
			page = q.Pagined.Page
		}
		pageSize = q.Pagined.PageSize
		// Contar total sin límite/offset
		countQB := qb.Session(&gorm.Session{})
		if err := countQB.Count(&total).Error; err != nil {
			return scdto.PaginatedResponse{}, err
		}
		if pageSize > 0 {
			totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
		}
		if page < 1 {
			page = 1
		}
		offset := (page - 1) * pageSize
		qb = qb.Limit(pageSize).Offset(offset)
	}

	results := make([]map[string]interface{}, 0)
	if err := qb.Find(&results).Error; err != nil {
		return scdto.PaginatedResponse{}, err
	}

	var resp scdto.PaginatedResponse
	resp.Data = results
	if pageSize > 0 {
		assignPaginationFields(&resp, page, pageSize, total, totalPages)
	}
	return resp, nil
}

// assignPaginationFields intenta asignar page, pageSize, total y totalPages si existen en el struct
func assignPaginationFields(resp interface{}, page, pageSize int, total int64, totalPages int) {
	rv := reflect.ValueOf(resp)
	if rv.Kind() != reflect.Ptr {
		return
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return
	}

	setInt := func(name string, v int) {
		f := rv.FieldByName(name)
		if f.IsValid() && f.CanSet() && (f.Kind() == reflect.Int || f.Kind() == reflect.Int64) {
			f.SetInt(int64(v))
		}
	}
	setInt64 := func(name string, v int64) {
		f := rv.FieldByName(name)
		if f.IsValid() && f.CanSet() && (f.Kind() == reflect.Int || f.Kind() == reflect.Int64) {
			f.SetInt(v)
		}
	}

	setInt("Page", page)
	setInt("PageSize", pageSize)
	setInt64("Total", total)
	setInt("TotalPages", totalPages)
}

func (r *FactoryRepositoryImpl) Update(table string, upd UpdateDTO) (int64, error) {
    fmt.Println("[Repo][Update] table:", table)
    fmt.Println("[Repo][Update] incoming payload:", upd)
    if err := r.validateIdent(table); err != nil {
        fmt.Println("[Repo][Update] validateIdent error:", err)
        return 0, err
    }
    if len(upd.Filters) == 0 {
        fmt.Println("[Repo][Update] no filters provided — refusing to update")
        return 0, errors.New("se requiere al menos un filtro para actualizar")
    }

	// Validar columnas a actualizar
    safeData := map[string]interface{}{}
    for k, v := range upd.Data {
        if err := r.validateIdent(k); err != nil {
            fmt.Println("[Repo][Update] invalid column name:", k, "err:", err)
            return 0, err
        }
        safeData[k] = v
    }
    if len(safeData) == 0 {
        fmt.Println("[Repo][Update] safeData empty — nothing to update")
        return 0, errors.New("no hay campos válidos para actualizar")
    }

    qb := r.DB.Table(table)
    q := QueryDTO{Filters: upd.Filters}
    fmt.Println("[Repo][Update] building query with filters:", q.Filters)
    qb, err := r.buildQuery(qb, q, table)
    if err != nil {
        fmt.Println("[Repo][Update] buildQuery error:", err)
        return 0, err
    }

    // Mapa de tipos de columnas
    colTypes, _ := r.getColumnTypes(table)
    fmt.Println("[Repo][Update] detected column types:", colTypes)

    // Acumular asignaciones para un único UPDATE
    updateMap := make(map[string]interface{})

	// Caso legacy: si existen columnas value y (opcional) tipo
    if val, ok := safeData["value"]; ok {
        if _, hasValueCol := colTypes["value"]; hasValueCol {
            var tipoForValidation string
            if _, hasTipoCol := colTypes["tipo"]; hasTipoCol {
                if tt, present := safeData["tipo"]; present {
                    if ts, ok := tt.(string); ok {
                        tipoForValidation = ts
                    }
                }
                if tipoForValidation == "" {
                    var count int64
                    if err := qb.Count(&count).Error; err == nil && count > 0 {
                        var tipos []string
                        qbTypes := r.DB.Table(table)
                        qbTypes, _ = r.buildQuery(qbTypes, q, table)
                        if err := qbTypes.Distinct().Pluck("tipo", &tipos).Error; err == nil {
                            uniq := make(map[string]struct{})
                            for _, t := range tipos {
                                if t != "" {
                                    uniq[t] = struct{}{}
                                }
                            }
                            if len(uniq) == 1 {
                                for k := range uniq {
                                    tipoForValidation = k
                                }
                            }
                            if len(uniq) > 1 {
                                fmt.Println("[Repo][Update] ambiguous 'value' update: multiple tipos detected; require 'tipo'")
                                return 0, errors.New("update con value ambiguo: múltiples tipos; incluye 'tipo'")
                            }
                        }
                    }
                }
                if tipoForValidation != "" {
                    if err := validateValueAgainstTipo(tipoForValidation, val); err != nil {
                        fmt.Println("[Repo][Update] value/tipo validation error:", err)
                        return 0, err
                    }
                }
            }
            // Preparar value como json/jsonb si corresponde (sin ejecutar aún)
            if t := strings.ToLower(colTypes["value"]); t == "jsonb" || t == "json" {
                b, err := json.Marshal(val)
                if err != nil {
                    fmt.Println("[Repo][Update] marshal value error:", err)
                    return 0, err
                }
                cast := "::jsonb"
                if t == "json" {
                    cast = "::json"
                }
                updateMap["value"] = gorm.Expr("?"+cast, string(b))
                delete(safeData, "value")
            }
        }
    }

    // Preparar otras columnas json/jsonb o normales en updateMap
    for k, v := range safeData {
        t := strings.ToLower(colTypes[k])
        if t == "json" || t == "jsonb" {
            b, err := json.Marshal(v)
            if err != nil {
                fmt.Println("[Repo][Update] marshal json field error for", k, ":", err)
                return 0, err
            }
            cast := "::jsonb"
            if t == "json" {
                cast = "::json"
            }
            updateMap[k] = gorm.Expr("?"+cast, string(b))
        } else {
            updateMap[k] = v
        }
    }

    if len(updateMap) == 0 {
        fmt.Println("[Repo][Update] nothing to update after processing — empty updateMap")
        return 0, errors.New("no hay campos válidos para actualizar")
    }

    fmt.Println("[Repo][Update] final updateMap for Updates:", updateMap)
    tx := qb.Updates(updateMap)
    fmt.Println("[Repo][Update] RowsAffected:", tx.RowsAffected, "Error:", tx.Error)
    return tx.RowsAffected, tx.Error
}

func (r *FactoryRepositoryImpl) DropTable(table string) error {
	if err := r.validateIdent(table); err != nil {
		return err
	}
	dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS \"%s\";", table)
	return r.DB.Exec(dropSQL).Error
}

// helpers
func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(v)
	}
}

func toStringSlice(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok {
		return []string{toString(v)}
	}
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		out = append(out, toString(e))
	}
	return out
}

func toLike(s string) string { return "%" + s + "%" }

// CreateRelation crea una relación FK entre dos tablas dinámicas y crea la columna en la segunda tabla.
func (r *FactoryRepositoryImpl) CreateRelation(rel RelationDTO) error {
	// Validar identificadores
	if err := r.validateIdent(rel.ReferencedTable); err != nil {
		return err
	}
	if err := r.validateIdent(rel.ReferencingTable); err != nil {
		return err
	}
	if err := r.validateIdent(rel.ColumnName); err != nil {
		return err
	}

	onDelete := strings.ToUpper(strings.TrimSpace(rel.OnDelete))
	onUpdate := strings.ToUpper(strings.TrimSpace(rel.OnUpdate))
	if onDelete == "" {
		onDelete = "NO ACTION"
	}
	if onUpdate == "" {
		onUpdate = "NO ACTION"
	}

	// Solo permitir valores válidos
	validActions := map[string]struct{}{"NO ACTION": {}, "CASCADE": {}, "RESTRICT": {}, "SET NULL": {}}
	if _, ok := validActions[onDelete]; !ok {
		return fmt.Errorf("on_delete inválido: %s", onDelete)
	}
	if _, ok := validActions[onUpdate]; !ok {
		return fmt.Errorf("on_update inválido: %s", onUpdate)
	}

	nullSQL := "NOT NULL"
	if rel.Nullable {
		nullSQL = "NULL"
	}

	// Nombre de la restricción
	constraint := fmt.Sprintf("fk_%s_%s_%s", rel.ReferencingTable, rel.ReferencedTable, rel.ColumnName)

	// Envolver en transacción
	return r.DB.Transaction(func(tx *gorm.DB) error {
		// 1) Crear columna si no existe (BIGINT para referenciar id BIGSERIAL)
		addCol := fmt.Sprintf("ALTER TABLE \"%s\" ADD COLUMN IF NOT EXISTS \"%s\" BIGINT %s;", rel.ReferencingTable, rel.ColumnName, nullSQL)
		if err := tx.Exec(addCol).Error; err != nil {
			return err
		}

		// 2) Agregar restricción FK si no existe (usando DO block)
		addFk := fmt.Sprintf(`DO $$
        BEGIN
            IF NOT EXISTS (
                SELECT 1 FROM information_schema.table_constraints tc
                WHERE tc.constraint_name = '%s'
                  AND tc.table_name = '%s'
                  AND tc.constraint_type = 'FOREIGN KEY'
            ) THEN
                ALTER TABLE "%s"
                ADD CONSTRAINT "%s" FOREIGN KEY ("%s")
                REFERENCES "%s"("id") ON DELETE %s ON UPDATE %s;
            END IF;
        END$$;`, constraint, rel.ReferencingTable, rel.ReferencingTable, constraint, rel.ColumnName, rel.ReferencedTable, onDelete, onUpdate)
		if err := tx.Exec(addFk).Error; err != nil {
			return err
		}

		// 3) (Opcional) Índice sobre la columna FK
		idxName := fmt.Sprintf("idx_%s_%s", rel.ReferencingTable, rel.ColumnName)
		addIdx := fmt.Sprintf("CREATE INDEX IF NOT EXISTS \"%s\" ON \"%s\"(\"%s\");", idxName, rel.ReferencingTable, rel.ColumnName)
		if err := tx.Exec(addIdx).Error; err != nil {
			return err
		}

		return nil
	})
}

// Insert inserta una fila validando columnas y manejando value como JSONB.
func (r *FactoryRepositoryImpl) Insert(table string, data map[string]interface{}) (uint64, error) {
	if err := r.validateIdent(table); err != nil {
		return 0, err
	}
	if data == nil || len(data) == 0 {
		return 0, errors.New("data vacía")
	}

	// Tipos de columnas para respetar estructura y castear JSON
	colTypes, _ := r.getColumnTypes(table)

	cols := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	args := make([]interface{}, 0, len(data))

	for k, v := range data {
		// bloquear id explícito en insert
		if k == "id" {
			continue
		}
		if err := r.validateIdent(k); err != nil {
			return 0, err
		}

		// Verificar que la columna exista
		ctype, ok := colTypes[k]
		if !ok {
			return 0, fmt.Errorf("columna no existe en tabla: %s", k)
		}

		cols = append(cols, fmt.Sprintf("\"%s\"", k))
		t := strings.ToLower(ctype)
		if t == "json" || t == "jsonb" {
			b, err := json.Marshal(v)
			if err != nil {
				return 0, err
			}
			if t == "jsonb" {
				placeholders = append(placeholders, "?::jsonb")
			} else {
				placeholders = append(placeholders, "?::json")
			}
			args = append(args, string(b))
		} else {
			placeholders = append(placeholders, "?")
			args = append(args, v)
		}
	}

	if len(cols) == 0 {
		return 0, errors.New("no hay columnas válidas para insertar")
	}

	// Intentar devolver PK si existe (no se asume nombre 'id')
	pkCol, pkType, _ := r.getPrimaryKeyInfo(table)
	if pkCol != "" {
		sql := fmt.Sprintf("INSERT INTO \"%s\" (%s) VALUES (%s) RETURNING \"%s\";", table, strings.Join(cols, ","), strings.Join(placeholders, ","), pkCol)
		// Tipos enteros soportados para retorno numérico
		switch pkType {
		case "bigint", "integer", "smallint", "serial", "bigserial":
			var id uint64
			if err := r.DB.Raw(sql, args...).Scan(&id).Error; err != nil {
				return 0, err
			}
			return id, nil
		default:
			var _dummy string
			if err := r.DB.Raw(sql, args...).Scan(&_dummy).Error; err != nil {
				return 0, err
			}
			return 0, nil
		}
	}
	// Sin PK conocida: ejecutar sin RETURNING
	execSQL := fmt.Sprintf("INSERT INTO \"%s\" (%s) VALUES (%s);", table, strings.Join(cols, ","), strings.Join(placeholders, ","))
	if err := r.DB.Exec(execSQL, args...).Error; err != nil {
		return 0, err
	}
	return 0, nil
}

// Delete borra filas por filtros; requiere al menos un filtro para evitar eliminación total accidental
func (r *FactoryRepositoryImpl) Delete(table string, q DeleteDTO) (int64, error) {
	if err := r.validateIdent(table); err != nil {
		return 0, err
	}
	if len(q.Filters) == 0 {
		return 0, errors.New("se requiere al menos un filtro para borrar")
	}

	qb := r.DB.Table(table)
	qread := QueryDTO{Filters: q.Filters}
	qb, err := r.buildQuery(qb, qread, table)
	if err != nil {
		return 0, err
	}
	tx := qb.Delete(nil)
	return tx.RowsAffected, tx.Error
}

// getColumnTypes retorna un mapa columna->tipo (preferentemente udt_name) en minúsculas
func (r *FactoryRepositoryImpl) getColumnTypes(table string) (map[string]string, error) {
	query := `SELECT column_name, COALESCE(NULLIF(udt_name,''), data_type) AS t
              FROM information_schema.columns
              WHERE table_name = ?`
	rows, err := r.DB.Raw(query, table).Rows()
	if err != nil {
		return map[string]string{}, err
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var name, t string
		if err := rows.Scan(&name, &t); err != nil {
			return out, err
		}
		out[name] = strings.ToLower(t)
	}
	return out, nil
}

// getPrimaryKeyInfo retorna el nombre y tipo de la primera columna PK (si existe)
func (r *FactoryRepositoryImpl) getPrimaryKeyInfo(table string) (string, string, error) {
	// Obtener nombre de la primera PK
	var pkCol string
	q := `SELECT kcu.column_name
          FROM information_schema.table_constraints tc
          JOIN information_schema.key_column_usage kcu
            ON tc.constraint_name = kcu.constraint_name
           AND tc.table_schema = kcu.table_schema
         WHERE tc.constraint_type = 'PRIMARY KEY'
           AND tc.table_name = ?
         ORDER BY kcu.ordinal_position
         LIMIT 1`
	rows, err := r.DB.Raw(q, table).Rows()
	if err != nil {
		return "", "", err
	}
	defer rows.Close()
	if rows.Next() {
		if err := rows.Scan(&pkCol); err != nil {
			return "", "", err
		}
	}
	if pkCol == "" {
		return "", "", nil
	}
	// Obtener tipo desde columnas
	colTypes, err := r.getColumnTypes(table)
	if err != nil {
		return pkCol, "", err
	}
	return pkCol, strings.ToLower(colTypes[pkCol]), nil
}

// validateSQLType permite una lista blanca de tipos con parámetros simples (varchar(n), numeric(p[,s]))
func validateSQLType(t string) (string, error) {
	s := strings.TrimSpace(strings.ToLower(t))
	base := s
	params := ""
	if i := strings.Index(s, "("); i != -1 && strings.HasSuffix(s, ")") {
		base = strings.TrimSpace(s[:i])
		params = strings.TrimSpace(s[i:])
		re := regexp.MustCompile(`^\(\s*\d+\s*(,\s*\d+\s*)?\)$`)
		if !re.MatchString(params) {
			return "", fmt.Errorf("parámetros de tipo inválidos: %s", params)
		}
	}
	allowed := map[string]bool{
		"smallint": true, "integer": true, "bigint": true,
		"serial": true, "bigserial": true,
		"real": true, "double precision": true,
		"numeric": true, "decimal": true,
		"varchar": true, "character varying": true,
		"char": true, "character": true,
		"text": true, "boolean": true,
		"date": true, "timestamp": true, "timestamp without time zone": true,
		"timestamp with time zone": true, "timestamptz": true,
		"time": true, "uuid": true,
		"json": true, "jsonb": true,
	}
	if !allowed[base] {
		return "", fmt.Errorf("tipo SQL no permitido: %s", base)
	}
	if params != "" {
		if !(base == "varchar" || base == "character varying" || base == "char" || base == "character" || base == "numeric" || base == "decimal") {
			return "", fmt.Errorf("el tipo %s no admite parámetros", base)
		}
	}
	return base + params, nil
}

// isSafeDefaultLiteral valida defaults simples: números, booleanos o strings entre comillas simples
func isSafeDefaultLiteral(v string) bool {
	s := strings.TrimSpace(v)
	if s == "true" || s == "false" {
		return true
	}
	numRe := regexp.MustCompile(`^[+-]?(\d+)(\.\d+)?$`)
	if numRe.MatchString(s) {
		return true
	}
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		return true
	}
	return false
}
