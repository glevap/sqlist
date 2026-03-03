package sqlist

import (
	"fmt"

	"github.com/Masterminds/squirrel"
)

// ============= МЕТОДЫ ПОСТРОЕНИЯ SQL =============

// buildBaseSelect создает базовый селект
func (b *SQLBuilder) buildBaseSelect() squirrel.SelectBuilder {
	selectBuilder := squirrel.Select(b.fields...).From(b.fromTable)

	// Добавляем JOIN
	for _, join := range b.joins {
		selectBuilder = selectBuilder.JoinClause(
			fmt.Sprintf("%s %s ON %s", join.Type, join.Table, join.Condition),
		)
	}

	// Добавляем WHERE условия!
	if len(b.pendingFilter) > 0 {
		selectBuilder = selectBuilder.Where(squirrel.And(b.applyPriority()))
	}

	// добавляем оставшиеся сырые условия
	if len(b.whereConditions) > 0 {
		selectBuilder = selectBuilder.Where(squirrel.And(b.whereConditions))
	}

	return selectBuilder
}

func (b *SQLBuilder) toCondition(cfg pendingFilter) squirrel.Sqlizer {
	switch cfg.Operator {
	case EQ, IN:
		return squirrel.Eq{cfg.DBField: cfg.Value}
	case NOT_EQ, NOT_IN:
		return squirrel.NotEq{cfg.DBField: cfg.Value}
	case LIKE:
		if fieldValue, ok := cfg.Value.(string); ok {
			return squirrel.Like{cfg.DBField: fieldValue + "%"}
		}
	case ILIKE:
		if fieldValue, ok := cfg.Value.(string); ok {
			return squirrel.ILike{cfg.DBField: "%" + fieldValue + "%"}
		}
	case BETWEEN:
		return squirrel.Expr(cfg.DBField+" BETWEEN ? AND ?", cfg.Args...)
	case GT:
		return squirrel.Gt{cfg.DBField: cfg.Value}
	case LT:
		return squirrel.Lt{cfg.DBField: cfg.Value}
	case GTE:
		return squirrel.GtOrEq{cfg.DBField: cfg.Value}
	case LTE:
		return squirrel.LtOrEq{cfg.DBField: cfg.Value}
	case IS_NULL:
		return squirrel.Eq{cfg.DBField: nil}
	case NOT_NULL:
		return squirrel.NotEq{cfg.DBField: nil}
	case EXPR:
		if expr, ok := cfg.Value.(string); ok {
			return squirrel.Expr(expr, cfg.Args...)
		}
	case EXPR_EQ:
		if expr, ok := cfg.Value.(string); ok {
			fullExpr := cfg.DBField + " = " + expr
			return squirrel.Expr(fullExpr, cfg.Args...)
		}
	}

	// todo: нужно переработать данный вариант
	// return nil

	return squirrel.Eq{"1": "1"}
}

// возвращаем слайс Sqlizer-объектов, отсортированных по приоритетности
func (b *SQLBuilder) applyPriority() []squirrel.Sqlizer {
	filters := make(map[string]pendingFilter, len(b.pendingFilter))

	for _, filter := range b.pendingFilter {
		filters[filter.DBField] = filter
	}

	var prioritized []squirrel.Sqlizer

	// если приоритетность не указана
	if len(b.priority) == 0 {
		for _, filter := range filters {
			prioritized = append(prioritized, b.toCondition(filter))
		}

		return prioritized
	}

	for _, item := range b.priority {
		item = b.mapField(item)
		if filter, ok := filters[item]; ok {
			prioritized = append(prioritized, b.toCondition(filter))
			delete(filters, item)
		}
	}

	for _, item := range filters {
		prioritized = append(prioritized, b.toCondition(item))
	}

	return prioritized
}

// BuildCount строит запрос для подсчета
func (b *SQLBuilder) BuildCount() (string, []any, error) {
	if len(b.whereConditions) > 0 || b.estimateTable == "" {
		selectBuilder := b.buildBaseSelect()

		countBuilder := squirrel.Select("COUNT(*)").FromSelect(selectBuilder, "subquery")

		return countBuilder.PlaceholderFormat(b.placeholder).ToSql()
	}

	// Приблизительный подсчет
	sql := "SELECT reltuples::bigint AS estimate FROM pg_class WHERE oid = $1::regclass"

	return sql, []interface{}{b.estimateTable}, nil
}

// BuildSelect строит запрос для выборки данных
func (b *SQLBuilder) BuildSelect() (string, []any, error) {
	selectBuilder := b.buildBaseSelect()

	// Добавляем сортировку
	if b.sort.Field != "" {
		selectBuilder = selectBuilder.OrderBy(b.sort.Field + " " + b.sort.Order)
	}

	// Добавляем пагинацию
	if b.limit > 0 {
		selectBuilder = selectBuilder.Limit(b.limit)
	}
	if b.offset > 0 {
		selectBuilder = selectBuilder.Offset(b.offset)
	}

	return selectBuilder.PlaceholderFormat(b.placeholder).ToSql()
}

// Reset сбрасывает состояние
func (b *SQLBuilder) Reset() *SQLBuilder {
	b.whereConditions = []squirrel.Sqlizer{}
	b.sort = SortConfig{}
	b.limit = 0
	b.offset = 0

	return b
}

// Clone создает копию с чистым состоянием
func (b *SQLBuilder) Clone() *SQLBuilder {
	return &SQLBuilder{
		fromTable:       b.fromTable,
		estimateTable:   b.estimateTable,
		fields:          append([]string{}, b.fields...),
		joins:           append([]joinConfig{}, b.joins...),
		placeholder:     b.placeholder,
		whereConditions: []squirrel.Sqlizer{},
		sort:            SortConfig{},
		limit:           0,
		offset:          0,
	}
}
