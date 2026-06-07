package sqlist

import (
	"errors"
	"fmt"
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

	// todo:
	// добавить проверку if field == "" и с trim

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
		b.err = fmt.Errorf("Builder error: поле %s не найдено в настройках полей запроса", field)
		return b
	}

	for _, cte := range b.cte {
		if cte == nil {
			continue
		}

		if slices.Contains(cte.fields, cfg.DBField) {
			cte.pendingFilter = append(cte.pendingFilter, b.withPending(cfg.Operator, cfg.DBField, value, args...))
			return b
		}
	}

	b.pendingFilter = append(b.pendingFilter,
		b.withPending(cfg.Operator, cfg.DBField, value, args...),
	)

	return b
}

// возвращает настроенный экземпляр CTE билдера
func (b *SQLBuilder) WithCTE(name string, alias string) *SQLBuilder {
	// b.cte = b.cloneForCTE()

	// необходимая защита
	// поскольку CTE создается через отдельный экземпляр билдера, то ему доступны
	// все методы, в том числе и WithCTE(), что будет являться нарушением синтаксиса SQL
	if b.withName != "" {
		b.err = errors.New("SQL Error: объявление CTE внутри CTE не поддерживается")
		return b
	}

	cte := b.cloneForCTE()
	cte.withName = name
	cte.alias = alias

	b.cte = append(b.cte, cte)

	// b.cte.withName = name

	/*
	 * это умышленное дублирование,
	 * т.к. в fromTable будет указан алиас/ы cte,
	 * но никак иначе (в случае нескольких CTE) мы не распределим фильтры
	 * по нужным секциям запроса,
	 * да и вытаскивать алиас из FROM небезопасно (он может быть не указан)
	 */
	// b.cte.alias = alias

	// return b.cte
	return cte
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
	if !slices.Contains([]string{"desc", "asc"}, strings.ToLower(order)) {
		b.err = fmt.Errorf("order contains unsupported expression: %s", order)
		return b
	}

	b.sort = SortConfig{Field: field, Order: order}
	return b
}

// SortIf устанавливает сортировку, если поле не пустое
// todo: а подразумевается, что должно быть condition, по которому будет применяться сортировочное правило!
func (b *SQLBuilder) SortIf(field, order string) *SQLBuilder {
	if field != "" {
		b.Sort(field, order)
	}
	return b
}

func (b *SQLBuilder) ApplySort(field, order string) *SQLBuilder {
	b.SortIf(b.mapField(field), order)

	if b.sort.Field != "" {
		b.cleanSortField()
	}
	return b
}

// при использовании настроек полей
// нужно получить очищенное имя поля сортировки,
// а поле в конфиге либо указано с алиасом,
// либо обернуто в выражение
// можно и нужно использовать при сортировке сырое имя поля,
// т.к. сортировка применяется к результату всего sql
func (b *SQLBuilder) cleanSortField() {
	field := extractInnerField(b.sort.Field)

	// Проверяем, не осталось ли запятых (признак нескольких аргументов)
	if strings.Contains(field, ",") {
		b.err = fmt.Errorf("sort field contains multiple arguments: %s", b.sort.Field)
		return
	}

	// Берём последнюю часть после точки
	if idx := strings.LastIndex(field, "."); idx != -1 {
		field = field[idx+1:]
	}

	b.sort.Field = strings.TrimSpace(field)

	// Финальная проверка: поле не должно содержать скобки
	if strings.ContainsAny(b.sort.Field, "()") {
		b.err = fmt.Errorf("sort field contains unsupported expression: %s", b.sort.Field)
	}
}

func extractInnerField(field string) string {
	field = strings.TrimSpace(field)

	// Ищем позицию после имени функции
	idx := strings.Index(field, "(")
	if idx == -1 {
		return field
	}

	// Ищем соответствующую закрывающую скобку
	depth := 0
	start := idx + 1

	for i := start; i < len(field); i++ {
		switch field[i] {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return extractInnerField(field[start:i])
			}

			depth--
		}
	}

	return field
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
