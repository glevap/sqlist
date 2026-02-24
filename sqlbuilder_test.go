package sqlist

import (
	"testing"

	"github.com/Masterminds/squirrel"
	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSQLBuilder(t *testing.T) {
	b := NewSQLBuilder()

	assert.NotNil(t, b)
	assert.Empty(t, b.fields)
	assert.Empty(t, b.joins)
	assert.Empty(t, b.whereConditions)
	assert.Equal(t, sq.Dollar, b.placeholder)
	assert.Equal(t, uint64(7), b.limit)
	assert.Equal(t, uint64(0), b.offset)
	assert.NotNil(t, b.fieldConfigs)
}

func TestWithPlaceholder(t *testing.T) {
	tests := []struct {
		name        string
		placeholder squirrel.PlaceholderFormat
	}{
		{"Question", sq.Question},
		{"Dollar", sq.Dollar},
		{"Colon", sq.Colon},
		{"AtP", sq.AtP},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewSQLBuilder().WithPlaceholder(tt.placeholder)
			assert.Equal(t, tt.placeholder, b.placeholder)
		})
	}
}

func TestWithFromAndEstimate(t *testing.T) {
	b := NewSQLBuilder().
		WithFrom("users").
		WithEstimate("users")

	assert.Equal(t, "users", b.fromTable)
	assert.Equal(t, "users", b.estimateTable)
}

func TestWithFields(t *testing.T) {
	t.Run("single field", func(t *testing.T) {
		b := NewSQLBuilder().WithField("id")
		assert.Equal(t, []string{"id"}, b.fields)
	})

	t.Run("multiple fields", func(t *testing.T) {
		b := NewSQLBuilder().WithFields("id", "name", "email")
		assert.Equal(t, []string{"id", "name", "email"}, b.fields)
	})

	t.Run("chained fields", func(t *testing.T) {
		b := NewSQLBuilder().
			WithField("id").
			WithField("name").
			WithFields("email", "created_at")

		assert.Equal(t, []string{"id", "name", "email", "created_at"}, b.fields)
	})
}

func TestJoinMethods(t *testing.T) {
	t.Run("inner join", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithInnerJoin("orders", "users.id = orders.user_id")

		assert.Len(t, b.joins, 1)
		assert.Equal(t, "JOIN", b.joins[0].Type)
		assert.Equal(t, "orders", b.joins[0].Table)
		assert.Equal(t, "users.id = orders.user_id", b.joins[0].Condition)
	})

	t.Run("left join", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithLeftJoin("profiles", "users.id = profiles.user_id")

		assert.Len(t, b.joins, 1)
		assert.Equal(t, "LEFT JOIN", b.joins[0].Type)
	})

	t.Run("right join", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithRightJoin("orders", "users.id = orders.user_id")

		assert.Len(t, b.joins, 1)
		assert.Equal(t, "RIGHT JOIN", b.joins[0].Type)
	})

	t.Run("full join", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithFullJoin("orders", "users.id = orders.user_id")

		assert.Len(t, b.joins, 1)
		assert.Equal(t, "FULL JOIN", b.joins[0].Type)
	})

	t.Run("multiple joins", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithInnerJoin("orders", "users.id = orders.user_id").
			WithLeftJoin("profiles", "users.id = profiles.user_id")

		assert.Len(t, b.joins, 2)
	})
}

func TestWhereConditions(t *testing.T) {
	b := NewSQLBuilder().WithFrom("users").WithFields("id", "name")

	t.Run("eq", func(t *testing.T) {
		b.Eq("id", 1)
		assert.Len(t, b.whereConditions, 1)
	})

	t.Run("eq if", func(t *testing.T) {
		b.EqIf(5, "age")
		assert.Len(t, b.whereConditions, 2)

		// nil value should not add condition
		b.EqIf(nil, "age")
		assert.Len(t, b.whereConditions, 2)
	})

	t.Run("not eq", func(t *testing.T) {
		b.NotEq("status", "deleted")
		assert.Len(t, b.whereConditions, 3)
	})

	t.Run("like", func(t *testing.T) {
		b.Like("name", "john")
		assert.Len(t, b.whereConditions, 4)

		// empty value should not add condition
		b.Like("name", "")
		assert.Len(t, b.whereConditions, 4)
	})

	t.Run("ilike", func(t *testing.T) {
		b.ILike("name", "doe")
		assert.Len(t, b.whereConditions, 5)
	})

	t.Run("in", func(t *testing.T) {
		b.In("id", []int{1, 2, 3})
		assert.Len(t, b.whereConditions, 6)
	})

	t.Run("between", func(t *testing.T) {
		b.Between("age", 18, 65)
		assert.Len(t, b.whereConditions, 7)

		// nil values should not add condition
		b.Between("age", nil, 65)
		assert.Len(t, b.whereConditions, 7)
	})

	t.Run("comparison operators", func(t *testing.T) {
		b.Gt("age", 18)
		b.Lt("age", 100)
		b.Gte("age", 21)
		b.Lte("age", 99)

		assert.Len(t, b.whereConditions, 11)
	})

	t.Run("null checks", func(t *testing.T) {
		b.IsNull("deleted_at")
		b.IsNotNull("email")

		assert.Len(t, b.whereConditions, 13)
	})
}

func TestWhereIf(t *testing.T) {
	b := NewSQLBuilder().WithFrom("users")

	// Should add condition
	b.WhereIf(true, squirrel.Eq{"active": true})
	assert.Len(t, b.whereConditions, 1)

	// Should not add condition
	b.WhereIf(false, squirrel.Eq{"deleted": false})
	assert.Len(t, b.whereConditions, 1)
}

func TestOrAndConditions(t *testing.T) {
	b := NewSQLBuilder().WithFrom("users")

	t.Run("or condition", func(t *testing.T) {
		cond1 := squirrel.Eq{"status": "active"}
		cond2 := squirrel.Eq{"status": "pending"}
		b.Or(cond1, cond2)

		assert.Len(t, b.whereConditions, 1) // Or группирует в одно условие
	})

	t.Run("and condition", func(t *testing.T) {
		cond1 := squirrel.Gt{"age": 18}
		cond2 := squirrel.Lt{"age": 65}
		b.And(cond1, cond2)

		assert.Len(t, b.whereConditions, 2) // And добавляет как отдельные условия (они объединятся в AND позже)
	})
}

func TestSorting(t *testing.T) {
	t.Run("sort without mapping", func(t *testing.T) {
		b := NewSQLBuilder().WithFrom("users")
		b.Sort("id", "DESC")
		assert.Equal(t, "id", b.sort.Field)
		assert.Equal(t, "DESC", b.sort.Order)
	})

	t.Run("sort with mapping", func(t *testing.T) {
		b := NewSQLBuilder().WithFrom("users").
			WithFieldConfig("user_id", "users.id", EQ)

		b.Sort("user_id", "ASC")
		assert.Equal(t, "users.id", b.sort.Field) // должно быть смаплено!
		assert.Equal(t, "ASC", b.sort.Order)
	})

	t.Run("sort if", func(t *testing.T) {
		b := NewSQLBuilder().WithFrom("users")

		// Should set sort
		b.SortIf("name", "ASC")
		assert.Equal(t, "name", b.sort.Field)

		// Should not change sort
		b.SortIf("", "DESC")
		assert.Equal(t, "name", b.sort.Field)
	})
}

func TestPagination(t *testing.T) {
	b := NewSQLBuilder()

	t.Run("limit and offset", func(t *testing.T) {
		b.Limit(10).Offset(5)
		assert.Equal(t, uint64(10), b.limit)
		assert.Equal(t, uint64(5), b.offset)
	})

	t.Run("limit zero", func(t *testing.T) {
		b.Limit(10)
		b.Limit(0) // should not change
		assert.Equal(t, uint64(10), b.limit)
	})

	t.Run("page", func(t *testing.T) {
		b.Page(3, 20)
		assert.Equal(t, uint64(20), b.limit)
		assert.Equal(t, uint64(40), b.offset) // (3-1)*20 = 40
	})

	t.Run("page zero", func(t *testing.T) {
		b.Page(0, 15)
		assert.Equal(t, uint64(15), b.limit)
		assert.Equal(t, uint64(0), b.offset) // page < 1 becomes 1: (1-1)*15 = 0
	})
}

func TestFieldConfigAndMapping(t *testing.T) {
	b := NewSQLBuilder()

	t.Run("add field config", func(t *testing.T) {
		b.WithFieldConfig("user_name", "users.name", EQ)

		cfg, ok := b.fieldConfigs["user_name"]
		assert.True(t, ok)
		assert.Equal(t, "users.name", cfg.DBField)
		assert.Equal(t, EQ, cfg.Operator)
	})

	t.Run("map field", func(t *testing.T) {
		mapped := b.mapField("user_name")
		assert.Equal(t, "users.name", mapped)

		// Unknown field returns as is
		mapped = b.mapField("unknown")
		assert.Equal(t, "unknown", mapped)
	})
}

func TestApplyFilter(t *testing.T) {
	b := NewSQLBuilder().WithFrom("users")

	// Setup field configs
	b.WithFieldConfig("name", "users.name", ILIKE)
	b.WithFieldConfig("age", "users.age", GT)
	b.WithFieldConfig("status", "users.status", EQ)

	t.Run("apply ilike filter", func(t *testing.T) {
		b.ApplyFilter("name", "john")
		assert.Len(t, b.whereConditions, 1)
	})

	t.Run("apply gt filter", func(t *testing.T) {
		b.ApplyFilter("age", "18")
		assert.Len(t, b.whereConditions, 2)
	})

	t.Run("apply eq filter", func(t *testing.T) {
		b.ApplyFilter("status", "active")
		assert.Len(t, b.whereConditions, 3)
	})

	t.Run("unknown field", func(t *testing.T) {
		b.ApplyFilter("unknown", "value")
		assert.Len(t, b.whereConditions, 3) // no change
	})

	t.Run("empty value", func(t *testing.T) {
		b.ApplyFilter("name", "")
		assert.Len(t, b.whereConditions, 3) // no change
	})
}

func TestBuildCount(t *testing.T) {
	t.Run("normal count with where", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithFields("id", "name").
			WithPlaceholder(sq.Dollar).
			Eq("active", true)

		sql, args, err := b.BuildCount()

		require.NoError(t, err)
		assert.Contains(t, sql, "SELECT COUNT(*)")
		assert.Contains(t, sql, "FROM")
		assert.Contains(t, sql, "WHERE")
		assert.Contains(t, sql, "active = $1") // проверяем, что условие есть
		assert.Len(t, args, 1)
		assert.Equal(t, true, args[0])
	})

	t.Run("estimate count", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithEstimate("users")

		sql, args, err := b.BuildCount()

		require.NoError(t, err)
		assert.Equal(t, "SELECT reltuples::bigint AS estimate FROM pg_class WHERE oid = $1::regclass", sql)
		assert.Len(t, args, 1)
		assert.Equal(t, "users", args[0])
	})

	t.Run("different placeholder", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithFields("id", "name"). // ← ЭТО БЫЛО ПРОПУЩЕНО!
			WithPlaceholder(sq.Question).
			Eq("active", true)

		sql, args, err := b.BuildCount()

		require.NoError(t, err)
		assert.Contains(t, sql, "SELECT COUNT(*)")
		assert.Contains(t, sql, "FROM")
		assert.Contains(t, sql, "?") // проверяем, что плейсхолдер вопроса есть
		assert.Len(t, args, 1)
	})
}

func TestBuildSelect(t *testing.T) {
	t.Run("basic select", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithFields("id", "name", "email")

		sql, args, err := b.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "SELECT id, name, email FROM users")
		assert.Empty(t, args)
	})

	t.Run("select with where", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithFields("id", "name").
			Eq("active", true).
			Like("name", "john")

		sql, args, err := b.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "WHERE")
		assert.Contains(t, sql, "active = $1")
		assert.Contains(t, sql, "name LIKE $2")
		assert.Len(t, args, 2)
	})

	t.Run("select with join", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithFields("users.id", "users.name", "orders.total").
			WithLeftJoin("orders", "users.id = orders.user_id")

		sql, _, err := b.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "LEFT JOIN orders ON users.id = orders.user_id")
	})

	t.Run("select with sort and pagination", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithFields("id", "name").
			Sort("name", "ASC").
			Limit(10).
			Offset(5)

		sql, _, err := b.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "ORDER BY name ASC")
		assert.Contains(t, sql, "LIMIT 10")
		assert.Contains(t, sql, "OFFSET 5")
	})

	t.Run("select with mapped sort", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithFields("id", "name").
			WithFieldConfig("user_name", "users.name", ILIKE).
			Sort("user_name", "DESC") // сортируем по алиасу

		sql, _, err := b.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "ORDER BY users.name DESC") // должен быть смаплен!
	})

	t.Run("question placeholder", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithFields("id", "name"). // ← И ЗДЕСЬ ТОЖЕ!
			WithPlaceholder(sq.Question).
			Eq("active", true)

		sql, args, err := b.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "SELECT id, name FROM users")
		assert.Contains(t, sql, "WHERE")
		assert.Contains(t, sql, "active = ?")
		assert.Len(t, args, 1)
	})
}

func TestResetAndClone(t *testing.T) {
	original := NewSQLBuilder().
		WithFrom("users").
		WithFields("id", "name").
		Eq("active", true).
		Sort("name", "ASC").
		Limit(10)

	t.Run("reset", func(t *testing.T) {
		clone := original.Clone()
		clone.Reset()

		assert.Empty(t, clone.whereConditions)
		assert.Empty(t, clone.sort.Field)
		assert.Equal(t, uint64(0), clone.limit)
		assert.Equal(t, uint64(0), clone.offset)

		// Config should remain
		assert.Equal(t, "users", clone.fromTable)
		assert.Equal(t, []string{"id", "name"}, clone.fields)
	})

	t.Run("clone", func(t *testing.T) {
		clone := original.Clone()

		// Should have same config
		assert.Equal(t, original.fromTable, clone.fromTable)
		assert.Equal(t, original.fields, clone.fields)
		assert.Equal(t, original.placeholder, clone.placeholder)

		// But empty state
		assert.Empty(t, clone.whereConditions)
		assert.Empty(t, clone.sort.Field)
		assert.Equal(t, uint64(0), clone.limit)

		// Modifying clone shouldn't affect original
		clone.Eq("status", "active")
		assert.Len(t, clone.whereConditions, 1)
		assert.Len(t, original.whereConditions, 1) // original still has its condition
	})
}

func TestComplexQuery(t *testing.T) {
	// Настраиваем маппинг полей
	b := NewSQLBuilder().
		WithFrom("users u").
		WithFields("u.id", "u.name", "o.total", "p.bio").
		WithLeftJoin("orders o", "u.id = o.user_id").
		WithLeftJoin("profiles p", "u.id = p.user_id").
		WithPlaceholder(sq.Dollar).
		WithFieldConfig("total", "o.total", GT).     // для фильтрации
		WithFieldConfig("name", "u.name", ILIKE).    // для фильтрации
		WithFieldConfig("total_sort", "o.total", GT) // для сортировки

	// Добавляем условия
	b.Eq("u.active", true)
	b.Gt("total", 100)           // используем алиас "total"
	b.ILike("name", "john")      // используем алиас "name"
	b.Sort("total_sort", "DESC") // сортируем по алиасу
	b.Limit(20)
	b.Offset(40)

	sql, args, err := b.BuildSelect()

	require.NoError(t, err)
	assert.Contains(t, sql, "FROM users u")
	assert.Contains(t, sql, "LEFT JOIN orders o ON u.id = o.user_id")
	assert.Contains(t, sql, "LEFT JOIN profiles p ON u.id = p.user_id")
	assert.Contains(t, sql, "WHERE")
	assert.Contains(t, sql, "u.active = $1")
	assert.Contains(t, sql, "o.total > $2")
	assert.Contains(t, sql, "u.name ILIKE $3")
	assert.Contains(t, sql, "ORDER BY o.total DESC") // должно быть смаплено!
	assert.Contains(t, sql, "LIMIT 20")
	assert.Contains(t, sql, "OFFSET 40")
	assert.Len(t, args, 3)
}

func TestApplyExpr(t *testing.T) {
	b := NewSQLBuilder().WithFrom("users")

	t.Run("apply expression", func(t *testing.T) {
		b.WithFieldConfig("snils", "toINT(snils)", EXPR_EQ)

		b.ApplyExpr("snils", "toINT(?)", 10)

		assert.Len(t, b.whereConditions, 1)

		sql, args, err := b.whereConditions[0].ToSql()

		assert.NoError(t, err)

		assert.Equal(t, "toINT(snils) = toINT(?)", sql)

		assert.Equal(t, []any{10}, args)
	})

	t.Run("apply expression with no EXPR_EQ", func(t *testing.T) {
		b.WithFieldConfig("snils", "toINT(snils)", EQ)

		b.ApplyExpr("snils", "toINT(?)", 10)

		assert.Len(t, b.whereConditions, 1)

		sql, args, err := b.whereConditions[0].ToSql()

		assert.NoError(t, err)

		assert.Equal(t, "toINT(snils) = ?", sql)

		assert.Equal(t, []any{10}, args)
	})
}
