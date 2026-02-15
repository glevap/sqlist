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
	if len(b.whereConditions) > 0 {
		selectBuilder = selectBuilder.Where(squirrel.And(b.whereConditions))
	}

	return selectBuilder
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
		sortField := b.sort.Field
		if mapped, ok := b.sortMapping[b.sort.Field]; ok {
			sortField = mapped
		}
		selectBuilder = selectBuilder.OrderBy(sortField + " " + b.sort.Order)
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
		sortMapping:     b.sortMapping,
		placeholder:     b.placeholder,
		whereConditions: []squirrel.Sqlizer{},
		sort:            SortConfig{},
		limit:           0,
		offset:          0,
	}
}
