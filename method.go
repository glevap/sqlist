package sqlist

import (
	"strings"

	"github.com/Masterminds/squirrel"
)

// МЕТОДЫ КОНФИГУРАЦИИ =====================================================================

// WithPlaceholder устанавливает формат плейсхолдеров
func (b *SQLBuilder) WithPlaceholder(placeholder squirrel.PlaceholderFormat) *SQLBuilder {
	b.placeholder = placeholder
	return b
}

// WithFrom устанавливает основную таблицу
func (b *SQLBuilder) WithFrom(table string) *SQLBuilder {
	b.fromTable = table
	return b
}

// WithEstimate устанавливает таблицу для приблизительного подсчета
func (b *SQLBuilder) WithEstimate(table string) *SQLBuilder {
	b.estimateTable = table
	return b
}

// WithField добавляет поле
func (b *SQLBuilder) WithField(field string) *SQLBuilder {
	b.fields = append(b.fields, field)
	return b
}

// WithFields добавляет несколько полей
func (b *SQLBuilder) WithFields(fields ...string) *SQLBuilder {
	b.fields = append(b.fields, fields...)
	return b
}

// WithFields добавляет несколько полей
func (b *SQLBuilder) WithPriority(fields string) *SQLBuilder {
	for _, field := range strings.Split(fields, ",") {
		b.priority = append(
			b.priority, strings.TrimSpace(field))
	}
	return b
}

// МЕТОДЫ ДЛЯ JOIN =========================================================================

// WithJoin добавляет произвольный JOIN
func (b *SQLBuilder) WithJoin(joinType, table, condition string, args ...interface{}) *SQLBuilder {
	// todo: нет проверки на тип JOIN
	// в случае некорректного типа SQL упадет
	b.joins = append(b.joins, joinConfig{
		Type:      joinType,
		Table:     table,
		Condition: condition,
		Args:      args,
	})
	return b
}

// WithLeftJoin добавляет LEFT JOIN
func (b *SQLBuilder) WithLeftJoin(table, condition string, args ...interface{}) *SQLBuilder {
	return b.WithJoin("LEFT JOIN", table, condition, args...)
}

// WithRightJoin добавляет RIGHT JOIN
func (b *SQLBuilder) WithRightJoin(table, condition string, args ...interface{}) *SQLBuilder {
	return b.WithJoin("RIGHT JOIN", table, condition, args...)
}

// WithInnerJoin добавляет INNER JOIN
func (b *SQLBuilder) WithInnerJoin(table, condition string, args ...interface{}) *SQLBuilder {
	return b.WithJoin("JOIN", table, condition, args...)
}

// WithFullJoin добавляет FULL JOIN
func (b *SQLBuilder) WithFullJoin(table, condition string, args ...interface{}) *SQLBuilder {
	return b.WithJoin("FULL JOIN", table, condition, args...)
}

// МЕТОДЫ ДЛЯ УСЛОВИЙ (ВСЕ ВОЗВРАЩАЮТ pendingFilter) =======================================

// todo: переделать
// Where добавляет произвольное условие
func (b *SQLBuilder) Where(condition squirrel.Sqlizer) *SQLBuilder {
	// 	b.whereConditions = append(b.whereConditions, condition)
	return b
}

// todo: переделать
// WhereIf добавляет условие, если флаг true
func (b *SQLBuilder) WhereIf(cond bool, condition squirrel.Sqlizer) *SQLBuilder {
	// if cond {
	// 	b.whereConditions = append(b.whereConditions, condition)
	// }
	return b
}

// унифицируем сохранение фильтров, ожидающих обработки и добавления в финальный SQL
func (b *SQLBuilder) withPending(op Op, field string, value any, args ...any) pendingFilter {
	return pendingFilter{
		FieldConfig: FieldConfig{
			DBField:  field,
			Operator: op,
		},
		Value: value,
		Args:  args,
	}
}

// Eq добавляет условие равенства
func (b *SQLBuilder) Eq(field string, value interface{}) *SQLBuilder {
	if value != nil {
		b.pendingFilter = append(b.pendingFilter,
			b.withPending(EQ, b.mapField(field), value),
		)
	}
	return b
}

// NotEq добавляет условие неравенства
func (b *SQLBuilder) NotEq(field string, value interface{}) *SQLBuilder {
	if value != nil {
		b.pendingFilter = append(b.pendingFilter,
			b.withPending(NOT_EQ, b.mapField(field), value),
		)
	}
	return b
}

// Like добавляет условие LIKE
func (b *SQLBuilder) Like(field string, value string) *SQLBuilder {
	if value != "" {
		b.pendingFilter = append(b.pendingFilter,
			b.withPending(LIKE, b.mapField(field), value+"%"),
		)
	}
	return b
}

// ILike добавляет условие ILIKE
func (b *SQLBuilder) ILike(field string, value string) *SQLBuilder {
	if value != "" {
		b.pendingFilter = append(b.pendingFilter,
			b.withPending(ILIKE, b.mapField(field), "%"+value+"%"),
		)
	}
	return b
}

// In добавляет условие IN
func (b *SQLBuilder) In(field string, values interface{}) *SQLBuilder {
	if values != nil {
		b.pendingFilter = append(b.pendingFilter,
			b.withPending(EQ, b.mapField(field), values),
		)
	}
	return b
}

// NotIn добавляет условие NOT IN
func (b *SQLBuilder) NotIn(field string, values interface{}) *SQLBuilder {
	if values != nil {
		b.pendingFilter = append(b.pendingFilter,
			b.withPending(NOT_EQ, b.mapField(field), values),
		)
	}
	return b
}

// Between добавляет условие BETWEEN
func (b *SQLBuilder) Between(field string, min, max interface{}) *SQLBuilder {
	if min != nil && max != nil {
		b.pendingFilter = append(b.pendingFilter,
			b.withPending(EXPR, b.mapField(field), b.mapField(field)+" BETWEEN ? AND ?", min, max),
		)
	}
	return b
}

// Gt добавляет условие "больше"
func (b *SQLBuilder) Gt(field string, value interface{}) *SQLBuilder {
	if value != nil {
		b.pendingFilter = append(b.pendingFilter,
			b.withPending(GT, b.mapField(field), value),
		)
	}
	return b
}

// Lt добавляет условие "меньше"
func (b *SQLBuilder) Lt(field string, value interface{}) *SQLBuilder {
	if value != nil {
		b.pendingFilter = append(b.pendingFilter,
			b.withPending(LT, b.mapField(field), value),
		)
	}
	return b
}

// Gte добавляет условие "больше или равно"
func (b *SQLBuilder) Gte(field string, value interface{}) *SQLBuilder {
	if value != nil {
		b.pendingFilter = append(b.pendingFilter,
			b.withPending(GTE, b.mapField(field), value),
		)
	}
	return b
}

// Lte добавляет условие "меньше или равно"
func (b *SQLBuilder) Lte(field string, value interface{}) *SQLBuilder {
	if value != nil {
		b.pendingFilter = append(b.pendingFilter,
			b.withPending(LTE, b.mapField(field), value),
		)
	}
	return b
}

// IsNull добавляет условие IS NULL
func (b *SQLBuilder) IsNull(field string) *SQLBuilder {
	b.pendingFilter = append(b.pendingFilter,
		b.withPending(EQ, b.mapField(field), nil),
	)
	return b
}

// IsNotNull добавляет условие IS NOT NULL
func (b *SQLBuilder) IsNotNull(field string) *SQLBuilder {
	b.pendingFilter = append(b.pendingFilter,
		b.withPending(NOT_EQ, b.mapField(field), nil),
	)
	return b
}

// Or группирует условия в OR
// func (b *SQLBuilder) Or(conditions ...squirrel.Sqlizer) *SQLBuilder {
// 	if len(conditions) > 0 {
// 		b.whereConditions = append(b.whereConditions, squirrel.Or(conditions))
// 	}
// 	return b
// }

// // And группирует условия в AND (обычно не нужно)
// func (b *SQLBuilder) And(conditions ...squirrel.Sqlizer) *SQLBuilder {
// 	if len(conditions) > 0 {
// 		b.whereConditions = append(b.whereConditions, squirrel.And(conditions))
// 	}
// 	return b
// }

func (b *SQLBuilder) Expr(field string, fullExpr string, args ...interface{}) *SQLBuilder {
	b.pendingFilter = append(b.pendingFilter,
		b.withPending(EXPR, field, fullExpr, nil),
	)

	return b
}

// ExprEq добавляет условие с функцией с правой стороны
// Пример: persons.snils2bcd64(snils) = persons.snils2bcd64('111-111-111 11')
func (b *SQLBuilder) ExprEq(leftField, rightExpr string, args ...interface{}) *SQLBuilder {
	fullExpr := leftField + " = " + rightExpr

	b.Expr(leftField, fullExpr, args...)

	return b
}

// МЕТОДЫ ДЛЯ СОРТИРОВКИ И ПАГИНАЦИИ =======================================================

// Sort устанавливает сортировку
func (b *SQLBuilder) Sort(field, order string) *SQLBuilder {
	b.sort = SortConfig{Field: b.mapField(field), Order: order}
	return b
}

// SortIf устанавливает сортировку, если поле не пустое
// todo: а подразумевается, что должно быть condition, по которому будет применяться сортировочное правило!
func (b *SQLBuilder) SortIf(field, order string) *SQLBuilder {
	if field != "" {
		b.sort = SortConfig{Field: b.mapField(field), Order: order}
	}
	return b
}

// Limit устанавливает лимит
func (b *SQLBuilder) Limit(limit uint64) *SQLBuilder {
	if limit == 0 {
		return b
	}

	b.limit = limit

	return b
}

// Offset устанавливает смещение
func (b *SQLBuilder) Offset(offset uint64) *SQLBuilder {
	b.offset = offset
	return b
}

// Page устанавливает номер страницы
func (b *SQLBuilder) Page(page, pageSize uint64) *SQLBuilder {
	if page < 1 {
		page = 1
	}
	b.limit = pageSize
	b.offset = (page - 1) * pageSize
	return b
}

func (b *SQLBuilder) WithFieldConfig(field string, dbField string, op Op) *SQLBuilder {
	if b.fieldConfigs == nil {
		b.fieldConfigs = make(map[string]FieldConfig)
	}

	b.fieldConfigs[field] = FieldConfig{
		DBField:  dbField,
		Operator: op,
	}

	return b
}

// ApplyFilter применяет фильтр. Удобно использовать для установки фильтров в цикле
func (b *SQLBuilder) ApplyFilter(field string, value string, args ...any) *SQLBuilder {
	if value == "" {
		return b
	}

	// todo:	если мои build-методы возвращают sql, args, err, то, можно писать ошибку!!!
	cfg, ok := b.fieldConfigs[field]
	if !ok {
		return b
	}

	b.pendingFilter = append(b.pendingFilter, b.withPending(cfg.Operator, cfg.DBField, value, args...))

	return b
}

// MapField возвращает настоящее имя колонки по псевдониму
func (b *SQLBuilder) mapField(alias string) string {
	if cfg, ok := b.fieldConfigs[alias]; ok {
		return cfg.DBField
	}
	return alias // если не нашли, возвращаем как есть
}
