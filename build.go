package sqlist

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Masterminds/squirrel"
	sq "github.com/Masterminds/squirrel"
)

// ============= МЕТОДЫ ПОСТРОЕНИЯ SQL =============

// buildBaseSelect создает базовый селект
func (b *SQLBuilder) buildBaseSelect() squirrel.SelectBuilder {
	selectBuilder := squirrel.Select(b.fields...).From(b.fromTable)

	for _, join := range b.joins {
		selectBuilder = selectBuilder.JoinClause(
			fmt.Sprintf("%s %s ON %s", join.Type, join.Table, join.Condition),
		)
	}

	// Добавляем WHERE условия!
	if len(b.pendingFilter) > 0 {
		selectBuilder = selectBuilder.Where(squirrel.And(b.toSQLizer()))
	}

	// если есть CTE, собираем по схеме Prefix().Suffix()
	if b.cte != nil {
		cteBuilder := b.cte.buildBaseSelect().Prefix(fmt.Sprintf("WITH %s AS (", b.cte.withName)).Suffix(")")

		sql, args, _ := cteBuilder.Prefix("").Suffix("").ToSql()

		selectBuilder = selectBuilder.Prefix(sql, args...)
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
			fullExpr := fmt.Sprintf("%s %s", cfg.DBField, expr)

			return squirrel.Expr(fullExpr, cfg.Args...)
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

func (b *SQLBuilder) toSQLizer() []squirrel.Sqlizer {
	sqlizers := make([]squirrel.Sqlizer, len(b.pendingFilter))

	for i, item := range b.pendingFilter {
		sqlizers[i] = b.toCondition(item)
	}

	return sqlizers
}

func (b *SQLBuilder) isSearchQuery() bool {
	if b.cte != nil {
		return len(b.cte.pendingFilter) > 0
	}

	return len(b.pendingFilter) > 0
}

func (b *SQLBuilder) BuildCount() (string, []any, error) {
	// либо ищем выборку по фильтру (да, org_ids []int тоже считается, либо не указали WithEstimate)
	if b.isSearchQuery() || b.estimateTable == "" {
		selectBuilder := b.buildBaseSelect()

		countBuilder := squirrel.Select("COUNT(*)").FromSelect(selectBuilder, "subquery")

		return countBuilder.PlaceholderFormat(b.placeholder).ToSql()
	}

	// получение данных из статистики postgres
	// неудачный хардкод, поскольку подразумевается использование любой поддерживаемой squirrel'ом БД
	sql := "SELECT reltuples::bigint AS estimate FROM pg_class WHERE oid = $1::regclass"

	return sql, []any{b.estimateTable}, nil
}

// BuildSelect строит запрос для выборки данных
func (b *SQLBuilder) BuildSelect() (string, []any, error) {
	selectBuilder := b.buildBaseSelect()

	// Добавляем сортировку
	if b.sort.Field != "" {
		var sortClause string

		// todo: если CTE несколько???
		if b.cte != nil {
			if slices.Contains(b.cte.fields, b.sort.Field) {
				fieldParts := strings.Split(b.sort.Field, ".")

				// это если указан alias
				if len(fieldParts) == 2 {
					sortClause = fmt.Sprintf("%s.%s  %s", b.cte.alias, fieldParts[1], b.sort.Order)
				}

				// а если не указан, будет неочевидность в sql при обращении к полям (SQLError при выполнении)
			}
		}

		// сортировка будет добавлена в корневой SELECT
		if sortClause != "" {
			selectBuilder = selectBuilder.OrderBy(sortClause)
		} else {
			selectBuilder = selectBuilder.OrderBy(fmt.Sprintf("%s %s", b.sort.Field, b.sort.Order))
		}
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

// Clone создает копию с чистым состоянием
func (b *SQLBuilder) cloneForCTE() *SQLBuilder {
	return &SQLBuilder{
		fields:        []string{},
		placeholder:   sq.Dollar, // по умолчанию PostgreSQL
		fieldConfigs:  b.fieldConfigs,
		pendingFilter: make([]pendingFilter, 0),
		joins:         []joinConfig{},
		limit:         7,
		offset:        0,
	}
}
