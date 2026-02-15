package sqlist

import (
	sq "github.com/Masterminds/squirrel"
)

var (
	// плейсхолдер, используемый в MySQL: ?
	Question = sq.Question

	// плейсхорлдер, используемый в PostgreSQL: $1, $2, $3
	Dollar = sq.Dollar

	// Позиционный плейсхолдер (:1, :2, :3).
	// Является промежуточным вариантом между $1 и ? и применяется для совместимости с некоторыми драйверами.
	Colon = sq.Colon

	// AtP is a PlaceholderFormat instance that replaces placeholders with
	// "@p"-prefixed positional placeholders (e.g. @p1, @p2, @p3).
	AtP = sq.AtP
)

type (
	// SQLBuilder строитель SQL запросов
	SQLBuilder struct {
		// Конфигурация
		fromTable     string
		estimateTable string
		fields        []string
		joins         []joinConfig
		sortMapping   map[string]string
		placeholder   sq.PlaceholderFormat
		fieldConfigs  map[string]FieldConfig

		// Состояние (все условия как Sqlizer)
		whereConditions []sq.Sqlizer
		sort            SortConfig
		limit           uint64
		offset          uint64
	}

	// FieldConfig описывает как обрабатывать поле
	FieldConfig struct {
		DBField  string // поле в БД
		Operator Op     // "eq", "like", "ilike", "gt", "lt"
	}

	// joinConfig использует Sqlizer для условия
	joinConfig struct {
		Type      string // "JOIN", "LEFT JOIN", "RIGHT JOIN"
		Table     string
		Condition string
		Args      interface{}
	}

	// SortConfig сортировка
	SortConfig struct {
		Field string
		Order string
	}

	// BuildResult результат построения запроса
	BuildResult struct {
		SQL  string
		Args []interface{}
		Err  error
	}

	Op string
)

const (
	EQ     Op = "eq"    // =
	NOT_EQ Op = "neq"   // !=
	LIKE   Op = "like"  // like
	ILIKE  Op = "ilike" // ilike
	GT     Op = "gt"    // >
	LT     Op = "lt"    // <
	GTE    Op = "gte"   // >=
	LTE    Op = "lte"   // <=
)

// ============= КОНСТРУКТОР =============

// NewSQLBuilder создает новый билдер с squirrel
func NewSQLBuilder() *SQLBuilder {
	return &SQLBuilder{
		fields:          []string{},
		joins:           []joinConfig{},
		sortMapping:     make(map[string]string),
		whereConditions: []sq.Sqlizer{},
		placeholder:     sq.Dollar, // по умолчанию PostgreSQL
		limit:           7,
		offset:          0,
		fieldConfigs:    make(map[string]FieldConfig),
	}
}
