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
		placeholder   sq.PlaceholderFormat
		fieldConfigs  map[string]FieldConfig

		// Состояние (все условия как Sqlizer)
		whereConditions []sq.Sqlizer
		sort            SortConfig
		limit           uint64
		offset          uint64

		// приоритетность полей по их именам: 0 - самый приоритетный
		priority      []string
		pendingFilter []pendingFilter
	}

	// FieldConfig описывает, как обрабатывать поле
	FieldConfig struct {
		DBField  string // имя поля в БД
		Operator Op     // "eq", "like", "ilike", "gt", "lt"
	}

	// joinConfig использует Sqlizer для условия
	joinConfig struct {
		Type      string // "JOIN", "LEFT JOIN", "RIGHT JOIN"
		Table     string
		Condition string
		Args      interface{}
	}

	// namedConditon сохраняет условие фильтрации с именем поля
	// для последующей приоритизации порядка полей в WHERE секции
	// финального запроса
	pendingFilter struct {
		FieldConfig
		Value any
		Args  []any
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
	EQ      Op = "eq"      // =
	NOT_EQ  Op = "neq"     // !=
	LIKE    Op = "like"    // like
	ILIKE   Op = "ilike"   // ilike
	GT      Op = "gt"      // >
	LT      Op = "lt"      // <
	GTE     Op = "gte"     // >=
	LTE     Op = "lte"     // <=
	EXPR    Op = "expr"    // expression
	EXPR_EQ Op = "expr_eq" // expression equal (specified)
)

// NewSQLBuilder создает новый билдер с squirrel
func NewSQLBuilder() *SQLBuilder {
	return &SQLBuilder{
		fields:          []string{},
		placeholder:     sq.Dollar, // по умолчанию PostgreSQL
		fieldConfigs:    make(map[string]FieldConfig),
		whereConditions: []sq.Sqlizer{},
		joins:           []joinConfig{},
		limit:           7,
		offset:          0,
		priority:        []string{},
		pendingFilter:   make([]pendingFilter, 0),
	}
}
