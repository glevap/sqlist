package sqlist

import (
	"slices"
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

	// алиас из FROM не вытащить, т.к. может быть более одного таргета в FROM-выражении
	// значит, пусть программист сам явно указывает алиасы при объеявлении CTE, а внешний Select
	// не будет иметь сохраненного алиаса

	return b
}

// WithEstimate устанавливает таблицу для приблизительного подсчета
func (b *SQLBuilder) WithEstimate(table string) *SQLBuilder {
	b.estimateTable = table
	return b
}

// WithField добавляет единичное поле
func (b *SQLBuilder) WithField(field string) *SQLBuilder {
	// можно включить после доработки
	// и сменить в BuildSelect/ApplyFilter функцию Contains на ==

	for _, val := range strings.Split(field, ",") {
		b.fields = append(b.fields, strings.TrimSpace(val))
	}

	return b
}

// настройки поля: исходное имя (из http-запроса), имя поля (с алиасом) в итоговом SQL
// и операнд, который будет применяться в WHERE-условии
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

	var isCTEFilter bool

	if b.cte != nil {
		if slices.Contains(b.cte.fields, cfg.DBField) {
			isCTEFilter = true
		}
	}

	// конфиг передается по частям, т.к. есть WHERE-метод, и он не связан с ApplyFilters
	if isCTEFilter {
		b.cte.pendingFilter = append(b.cte.pendingFilter,
			b.withPending(cfg.Operator, cfg.DBField, value, args...),
		)
	} else {
		b.pendingFilter = append(b.pendingFilter,
			b.withPending(cfg.Operator, cfg.DBField, value, args...),
		)
	}

	return b
}

// возвращает настроенный экземпляр CTE билдера
func (b *SQLBuilder) WithCTE(name string, alias string) *SQLBuilder {
	b.cte = b.cloneForCTE()

	b.cte.withName = name

	/*
	 * это умышленное дублирование,
	 * т.к. в fromTable будет указан алиас/ы cte,
	 * но никак иначе (в случае нескольких CTE) мы не распределим фильтры
	 * по нужным секциям запроса,
	 * да и вытаскивать алиас из FROM небезопасно (он может быть не указан)
	 */
	b.cte.alias = alias

	return b.cte
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

// MapField возвращает настоящее имя колонки по псевдониму
func (b *SQLBuilder) mapField(alias string) string {
	if cfg, ok := b.fieldConfigs[alias]; ok {
		return cfg.DBField
	}
	return alias // если не нашли, возвращаем как есть
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

// Where добавляет произвольное условие
func (b *SQLBuilder) Where(op Op, field string, value any, args ...any) *SQLBuilder {
	b.pendingFilter = append(b.pendingFilter, b.withPending(op, field, value, args...))

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
