package sqlist

import (
	"strings"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSQLBuilder(t *testing.T) {
	b := NewSQLBuilder()

	assert.NotNil(t, b)
	assert.Equal(t, squirrel.Dollar, b.placeholder)
	assert.Empty(t, b.fields)
	assert.NotNil(t, b.fieldConfigs)
	assert.NotNil(t, b.pendingFilter)
	assert.Empty(t, b.joins)
	assert.Equal(t, uint64(7), b.limit)
	assert.Equal(t, uint64(0), b.offset)
	assert.Empty(t, b.estimateTable)
	assert.Empty(t, b.fromTable)
	assert.Nil(t, b.err)
}

func TestWithPlaceholder(t *testing.T) {
	tests := []struct {
		name        string
		placeholder squirrel.PlaceholderFormat
	}{
		{"Question", squirrel.Question},
		{"Dollar", squirrel.Dollar},
		{"Colon", squirrel.Colon},
		{"AtP", squirrel.AtP},
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

func TestWithField(t *testing.T) {
	t.Run("single field", func(t *testing.T) {
		b := NewSQLBuilder().WithField("id")
		assert.Equal(t, []string{"id"}, b.fields)
	})

	t.Run("multiple fields via comma", func(t *testing.T) {
		b := NewSQLBuilder().WithField("id,name,email")
		assert.Equal(t, []string{"id", "name", "email"}, b.fields)
	})

	t.Run("chained fields", func(t *testing.T) {
		b := NewSQLBuilder().
			WithField("id").
			WithField("name").
			WithField("email,created_at")

		assert.Equal(t, []string{"id", "name", "email", "created_at"}, b.fields)
	})

	t.Run("fields with spaces", func(t *testing.T) {
		b := NewSQLBuilder().WithField("id, name, email")
		assert.Equal(t, []string{"id", "name", "email"}, b.fields)
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
		assert.Equal(t, "ON users.id = orders.user_id", b.joins[0].Condition)
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
		assert.Equal(t, "JOIN", b.joins[0].Type)
		assert.Equal(t, "LEFT JOIN", b.joins[1].Type)
	})

	t.Run("custom join", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithJoin("CROSS JOIN", "roles", "1=1")

		assert.Len(t, b.joins, 1)
		assert.Equal(t, "CROSS JOIN", b.joins[0].Type)
	})

	// Join methods have been updated to allow the use of the USING operator in the WithJoin method.

	t.Run("join clause with default ON clause", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users u").
			WithField("u.id, u.test, orders.email").
			WithFullJoin("orders", "u.id = orders.user_id")

		sql, _, err := b.BuildSelect()

		assert.NoError(t, err)
		assert.Contains(t, sql, "FULL JOIN orders ON u.id")
	})

	t.Run("join clause with default ON clause", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users u").
			WithField("u.id, u.test, orders.email").
			WithJoin("LEFT JOIN", "orders", "USING(id)")

		sql, _, err := b.BuildSelect()

		assert.NoError(t, err)
		assert.Contains(t, sql, "LEFT JOIN orders USING(id)")
	})

}

func TestWithFieldConfig(t *testing.T) {
	b := NewSQLBuilder()

	b.WithFieldConfig("user_name", "users.name", ILIKE)
	b.WithFieldConfig("user_age", "users.age", GT)

	cfg, ok := b.fieldConfigs["user_name"]
	assert.True(t, ok)
	assert.Equal(t, "users.name", cfg.DBField)
	assert.Equal(t, ILIKE, cfg.Operator)

	cfg, ok = b.fieldConfigs["user_age"]
	assert.True(t, ok)
	assert.Equal(t, "users.age", cfg.DBField)
	assert.Equal(t, GT, cfg.Operator)
}

func TestMapField(t *testing.T) {
	b := NewSQLBuilder().
		WithFieldConfig("user_name", "users.name", ILIKE)

	mapped := b.mapField("user_name")
	assert.Equal(t, "users.name", mapped)

	// Unknown field returns as is
	mapped = b.mapField("unknown")
	assert.Equal(t, "unknown", mapped)
}

func TestApplyFilter(t *testing.T) {
	b := NewSQLBuilder().
		WithFrom("users").
		WithFieldConfig("name", "users.name", ILIKE).
		WithFieldConfig("age", "users.age", GT).
		WithFieldConfig("status", "users.status", EQ)

	t.Run("apply filter with value", func(t *testing.T) {
		b2 := b.ApplyFilter("name", "john")
		assert.Len(t, b2.pendingFilter, 1)
		assert.Equal(t, "users.name", b2.pendingFilter[0].DBField)
		assert.Equal(t, ILIKE, b2.pendingFilter[0].Operator)
		assert.Equal(t, "john", b2.pendingFilter[0].Value)
	})

	t.Run("apply filter with empty value", func(t *testing.T) {
		b3 := NewSQLBuilder().
			WithFrom("users").
			WithFieldConfig("name", "users.name", ILIKE)

		b3.ApplyFilter("name", "")
		assert.Len(t, b3.pendingFilter, 0) // empty value should be ignored
	})

	t.Run("apply filter with unknown field", func(t *testing.T) {
		b4 := NewSQLBuilder().
			WithFrom("users").
			WithFieldConfig("name", "users.name", ILIKE)

		b4.ApplyFilter("unknown", "value")
		assert.NotNil(t, b4.err)
		assert.Contains(t, b4.err.Error(), "не найдено в настройках")
	})
}

func TestWhereMethod(t *testing.T) {
	b := NewSQLBuilder().
		WithFrom("users").
		WithField("id,name")

	b.Where(EQ, "users.id", 1)
	b.Where(LIKE, "users.name", "john%")

	assert.Len(t, b.pendingFilter, 2)
	assert.Equal(t, EQ, b.pendingFilter[0].Operator)
	assert.Equal(t, "users.id", b.pendingFilter[0].DBField)
	assert.Equal(t, 1, b.pendingFilter[0].Value)
	assert.Equal(t, LIKE, b.pendingFilter[1].Operator)
	assert.Equal(t, "users.name", b.pendingFilter[1].DBField)
}

func TestSorting(t *testing.T) {
	t.Run("sort without mapping", func(t *testing.T) {
		b := NewSQLBuilder().WithFrom("users")
		b.Sort("id", "DESC")
		assert.Equal(t, "id", b.sort.Field)
		assert.Equal(t, "DESC", b.sort.Order)
	})

	t.Run("sort with mapping", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithFieldConfig("user_id", "users.id", EQ)

		b.Sort("user_id", "ASC")
		assert.Equal(t, "user_id", b.sort.Field)
		assert.Equal(t, "ASC", b.sort.Order)
	})

	t.Run("sort if with condition", func(t *testing.T) {
		b := NewSQLBuilder().WithFrom("users")

		// Should set sort
		b.SortIf("name", "ASC")
		assert.Equal(t, "name", b.sort.Field)

		// Should not change sort when field is empty
		b.SortIf("", "DESC")
		assert.Equal(t, "name", b.sort.Field)
	})
}

func TestPagination(t *testing.T) {
	// b := NewSQLBuilder()

	t.Run("limit and offset", func(t *testing.T) {
		b2 := NewSQLBuilder()
		b2.Limit(10).Offset(5)
		assert.Equal(t, uint64(10), b2.limit)
		assert.Equal(t, uint64(5), b2.offset)
	})

	t.Run("limit zero should be ignored", func(t *testing.T) {
		b2 := NewSQLBuilder()
		b2.Limit(10)
		b2.Limit(0)
		assert.Equal(t, uint64(10), b2.limit) // should not change
	})

	t.Run("page calculation", func(t *testing.T) {
		b2 := NewSQLBuilder()
		b2.Page(3, 20)
		assert.Equal(t, uint64(20), b2.limit)
		assert.Equal(t, uint64(40), b2.offset) // (3-1)*20 = 40
	})

	t.Run("page zero becomes page 1", func(t *testing.T) {
		b2 := NewSQLBuilder()
		b2.Page(0, 15)
		assert.Equal(t, uint64(15), b2.limit)
		assert.Equal(t, uint64(0), b2.offset)
	})
}

func TestBuildSelect(t *testing.T) {
	t.Run("basic select", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithField("id").
			WithField("name").
			WithField("email")

		sql, args, err := b.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "SELECT id, name, email FROM users")
		assert.Empty(t, args)
	})

	t.Run("select with WHERE conditions", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithField("id, name").
			Where(EQ, "active", true).
			Where(LIKE, "name", "john%") // todo

		sql, args, err := b.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "WHERE")
		assert.Contains(t, sql, "active = $1")
		assert.Contains(t, sql, "name LIKE $2")
		assert.Len(t, args, 2)
		assert.Equal(t, true, args[0])
		assert.Equal(t, "john%", args[1])
	})

	t.Run("select with JOIN", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithField("users.id, users.name, orders.total").
			WithLeftJoin("orders", "users.id = orders.user_id")

		sql, _, err := b.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "LEFT JOIN orders ON users.id = orders.user_id")
	})

	t.Run("select with sorting and pagination", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithField("id, name").
			Sort("name", "ASC").
			Limit(10).
			Offset(5)

		sql, _, err := b.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "ORDER BY name ASC")
		assert.Contains(t, sql, "LIMIT 10")
		assert.Contains(t, sql, "OFFSET 5")
	})

	t.Run("select with question placeholder", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithField("id, name").
			WithPlaceholder(squirrel.Question).
			Where(EQ, "active", true)

		sql, args, err := b.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "active = ?")
		assert.Len(t, args, 1)
	})

	t.Run("select with error - no from table", func(t *testing.T) {
		b := NewSQLBuilder().WithField("id")
		// No WithFrom called

		_, _, err := b.BuildSelect()
		// squirrel will generate SQL without FROM clause which is invalid
		// but our builder doesn't validate this - it's a known limitation
		assert.NoError(t, err) // squirrel allows this
	})
}

func TestFiltersWithLIKE(t *testing.T) {
	t.Run("select with filter LIKE", func(t *testing.T) {
		b := NewSQLBuilder().WithPlaceholder(Dollar).WithFrom("users u").
			WithField("*").
			WithFieldConfig("name", "u.name", LIKE)

		name := "john"

		b.ApplyFilter("name", name)

		sql, args, err := b.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "SELECT * FROM users u WHERE (u.name LIKE $1)")
		assert.Contains(t, args, "john")
	})

	t.Run("select with filter LIKE_WITH_START", func(t *testing.T) {
		b := NewSQLBuilder().WithPlaceholder(Dollar).WithFrom("users u").
			WithField("*").
			WithFieldConfig("name", "u.name", LIKE_WITH_START)

		name := "john"

		b.ApplyFilter("name", name)

		sql, args, err := b.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "SELECT * FROM users u WHERE (u.name LIKE $1)")
		assert.Contains(t, args, "%john")
	})

	t.Run("select with filter LIKE_WITH_END", func(t *testing.T) {
		b := NewSQLBuilder().WithPlaceholder(Dollar).WithFrom("users u").
			WithField("*").
			WithFieldConfig("name", "u.name", LIKE_WITH_END)

		name := "john"

		b.ApplyFilter("name", name)

		sql, args, err := b.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "SELECT * FROM users u WHERE (u.name LIKE $1)")
		assert.Contains(t, args, "john%")
	})

	t.Run("select with filter LIKE_WITH_FULL", func(t *testing.T) {
		b := NewSQLBuilder().WithPlaceholder(Dollar).WithFrom("users u").
			WithField("*").
			WithFieldConfig("name", "u.name", LIKE_WITH_FULL)

		name := "john"

		b.ApplyFilter("name", name)

		sql, args, err := b.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "SELECT * FROM users u WHERE (u.name LIKE $1)")
		assert.Contains(t, args, "%john%")
	})
}

func TestFiltersWithILIKE(t *testing.T) {
	t.Run("select with filter ILIKE", func(t *testing.T) {
		b := NewSQLBuilder().WithPlaceholder(Dollar).WithFrom("users u").
			WithField("*").
			WithFieldConfig("name", "u.name", ILIKE)

		name := "john"

		b.ApplyFilter("name", name)

		sql, args, err := b.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "SELECT * FROM users u WHERE (u.name ILIKE $1)")
		assert.Contains(t, args, "john")
	})

	t.Run("select with filter ILIKE_WITH_START", func(t *testing.T) {
		b := NewSQLBuilder().WithPlaceholder(Dollar).WithFrom("users u").
			WithField("*").
			WithFieldConfig("name", "u.name", ILIKE_WITH_START)

		name := "john"

		b.ApplyFilter("name", name)

		sql, args, err := b.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "SELECT * FROM users u WHERE (u.name ILIKE $1)")
		assert.Contains(t, args, "%john")
	})

	t.Run("select with filter ILIKE_WITH_END", func(t *testing.T) {
		b := NewSQLBuilder().WithPlaceholder(Dollar).WithFrom("users u").
			WithField("*").
			WithFieldConfig("name", "u.name", ILIKE_WITH_END)

		name := "john"

		b.ApplyFilter("name", name)

		sql, args, err := b.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "SELECT * FROM users u WHERE (u.name ILIKE $1)")
		assert.Contains(t, args, "john%")
	})

	t.Run("select with filter ILIKE_WITH_FULL", func(t *testing.T) {
		b := NewSQLBuilder().WithPlaceholder(Dollar).WithFrom("users u").
			WithField("*").
			WithFieldConfig("name", "u.name", ILIKE_WITH_FULL)

		name := "john"

		b.ApplyFilter("name", name)

		sql, args, err := b.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "SELECT * FROM users u WHERE (u.name ILIKE $1)")
		assert.Contains(t, args, "%john%")
	})
}

func TestBuildCount(t *testing.T) {
	t.Run("normal count with WHERE conditions", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithField("id, name").
			Where(EQ, "active", true)

		sql, args, err := b.BuildCount()

		require.NoError(t, err)
		assert.Contains(t, sql, "SELECT COUNT(*)")
		assert.Contains(t, sql, "FROM")
		assert.Contains(t, sql, "WHERE")
		assert.Contains(t, sql, "active = $1")
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

	t.Run("estimate count with question placeholder", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithEstimate("users").
			WithPlaceholder(squirrel.Question)

		sql, args, err := b.BuildCount()

		require.NoError(t, err)
		// Estimate query uses PostgreSQL syntax regardless of placeholder
		assert.Contains(t, sql, "$1")
		assert.Len(t, args, 1)
	})

	t.Run("count with no conditions and no estimate", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithField("id, name")

		sql, args, err := b.BuildCount()

		require.NoError(t, err)
		assert.Contains(t, sql, "SELECT COUNT(*)")
		assert.Contains(t, sql, "FROM users")
		assert.Empty(t, args)
	})
}

func TestConditionOperators(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(b *SQLBuilder)
		expected  string
	}{
		{
			name: "EQ",
			setupFunc: func(b *SQLBuilder) {
				b.Where(EQ, "status", "active")
			},
			expected: "status = $1",
		},
		{
			name: "NOT_EQ",
			setupFunc: func(b *SQLBuilder) {
				b.Where(NOT_EQ, "status", "deleted")
			},
			expected: "status <> $1",
		},
		{
			name: "GT",
			setupFunc: func(b *SQLBuilder) {
				b.Where(GT, "age", 18)
			},
			expected: "age > $1",
		},
		{
			name: "LT",
			setupFunc: func(b *SQLBuilder) {
				b.Where(LT, "age", 100)
			},
			expected: "age < $1",
		},
		{
			name: "GTE",
			setupFunc: func(b *SQLBuilder) {
				b.Where(GTE, "age", 18)
			},
			expected: "age >= $1",
		},
		{
			name: "LTE",
			setupFunc: func(b *SQLBuilder) {
				b.Where(LTE, "age", 65)
			},
			expected: "age <= $1",
		},
		{
			name: "LIKE",
			setupFunc: func(b *SQLBuilder) {
				b.Where(LIKE, "name", "john")
			},
			expected: "name LIKE $1",
		},
		{
			name: "ILIKE",
			setupFunc: func(b *SQLBuilder) {
				b.Where(ILIKE, "name", "john")
			},
			expected: "name ILIKE $1",
		},
		{
			name: "IN",
			setupFunc: func(b *SQLBuilder) {
				b.Where(IN, "id", []int{1, 2, 3})
			},
			expected: "id IN ($1,$2,$3)",
		},
		{
			name: "NOT_IN",
			setupFunc: func(b *SQLBuilder) {
				b.Where(NOT_IN, "status", []string{"deleted", "banned"})
			},
			expected: "status NOT IN ($1,$2)",
		},
		{
			name: "IS_NULL",
			setupFunc: func(b *SQLBuilder) {
				b.Where(IS_NULL, "deleted_at", nil)
			},
			expected: "deleted_at IS NULL",
		},
		{
			name: "NOT_NULL",
			setupFunc: func(b *SQLBuilder) {
				b.Where(NOT_NULL, "email", nil)
			},
			expected: "email IS NOT NULL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewSQLBuilder().
				WithFrom("users").
				WithField("id")

			tt.setupFunc(b)

			sql, _, err := b.BuildSelect()
			require.NoError(t, err)
			assert.Contains(t, sql, tt.expected)
		})
	}
}

func TestBETWEENOperator(t *testing.T) {
	t.Run("date filter with BETWEEN operator in WHERE", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithField("id").
			Where(BETWEEN, "age", nil, 18, 65).
			WithPlaceholder(Question)

		sql, args, err := b.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "age BETWEEN ? AND ?")
		assert.Len(t, args, 2)
		assert.Equal(t, 18, args[0])
		assert.Equal(t, 65, args[1])
	})

	t.Run("date filter with BETWEEN operator in WithFieldConfig", func(t *testing.T) {
		main := NewSQLBuilder().
			WithFieldConfig("created", "c1.created_at", BETWEEN).
			WithFrom("final").
			WithField("*")

		main.WithCTE("cte1", "c1").
			WithFrom("table1").
			WithField("id")

		d := time.Now()

		// need startDate & finalDate (+1 Day to startDate)
		main.ApplyFilter("created", "?", d, d.AddDate(0, 0, 1).Truncate(24*time.Hour))

		sql, args, err := main.BuildSelect()

		assert.NoError(t, err)
		assert.Len(t, args, 2)
		assert.Contains(t, sql, "BETWEEN $1 AND $2")
	})
}

func TestComplexQuery(t *testing.T) {
	b := NewSQLBuilder().
		WithFrom("users u").
		WithField("u.id").
		WithField("u.name").
		WithField("o.total").
		WithLeftJoin("orders o", "u.id = o.user_id").
		WithLeftJoin("profiles p", "u.id = p.user_id").
		WithFieldConfig("total", "o.total", GT).
		WithFieldConfig("name", "u.name", ILIKE).
		WithPlaceholder(squirrel.Dollar)

	// Add conditions
	b.Where(EQ, "u.active", true)
	b.Where(GT, "o.total", 100)

	// without explicitly specifying a placeholder, it will be john
	b.Where(ILIKE, "u.name", "john")

	b.Sort("o.total", "DESC")
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
	assert.Contains(t, sql, "ORDER BY o.total DESC")
	assert.Contains(t, sql, "LIMIT 20")
	assert.Contains(t, sql, "OFFSET 40")
	assert.Len(t, args, 3)
	assert.Equal(t, true, args[0])
	assert.Equal(t, 100, args[1])
	assert.Equal(t, "john", args[2])
}

func TestCTE(t *testing.T) {
	t.Run("simple CTE", func(t *testing.T) {
		// Create main builder
		main := NewSQLBuilder().
			WithFrom("active_users").
			WithField("id, name, email")

		// Create CTE
		cte := main.WithCTE("active_users_cte", "au")
		cte.WithFrom("users").
			WithField("id, name, email").
			Where(EQ, "active", true)

		sql, args, err := main.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "WITH active_users_cte AS (")
		assert.Contains(t, sql, "SELECT id, name, email FROM users")
		assert.Contains(t, sql, "WHERE (active = $1)")
		assert.Contains(t, sql, "SELECT id, name, email FROM active_users")
		assert.Len(t, args, 1)
		assert.Equal(t, true, args[0])
	})

	t.Run("CTE with nested CTE not allowed", func(t *testing.T) {
		main := NewSQLBuilder()

		cte := main.WithCTE("level1", "l1")
		cte.WithFrom("table1")

		// Trying to create CTE inside CTE should return error
		innerCTE := cte.WithCTE("level2", "l2")
		assert.NotNil(t, innerCTE.err)
		assert.Contains(t, innerCTE.err.Error(), "не поддерживается")
	})

	t.Run("multiple CTEs", func(t *testing.T) {
		main := NewSQLBuilder().
			WithFrom("final_data").
			WithField("*")

		// First CTE
		cte1 := main.WithCTE("active_users", "au")
		cte1.WithFrom("users").
			WithField("id, name").
			Where(EQ, "active", true)

		// Second CTE
		cte2 := main.WithCTE("recent_orders", "ro")
		cte2.WithFrom("orders").
			WithField("user_id, total").
			Where(GT, "created_at", "2024-01-01")

		sql, _, err := main.BuildSelect()

		require.NoError(t, err)
		assert.Contains(t, sql, "WITH active_users AS (")
		assert.Contains(t, sql, ", recent_orders AS (")
	})
}

func TestErrorHandling(t *testing.T) {
	t.Run("apply filter with unknown field sets error", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithFieldConfig("known_field", "db.known", EQ)

		b.ApplyFilter("unknown_field", "value")

		assert.NotNil(t, b.err)
		assert.Contains(t, b.err.Error(), "не найдено в настройках")
	})

	t.Run("BuildCount returns error when builder has error", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithFieldConfig("known_field", "db.known", EQ)

		b.ApplyFilter("unknown_field", "value")
		_, _, err := b.BuildCount()

		assert.Error(t, err)
	})

	t.Run("BuildSelect returns error when builder has error", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithFieldConfig("known_field", "db.known", EQ)

		b.ApplyFilter("unknown_field", "value")
		_, _, err := b.BuildSelect()

		assert.Error(t, err)
	})
}

func TestCloneForCTE(t *testing.T) {
	original := NewSQLBuilder().
		WithFrom("users").
		WithField("id, name").
		WithPlaceholder(squirrel.Question).
		WithFieldConfig("user_name", "users.name", ILIKE)

	clone := original.cloneForCTE()

	// Clone should have fresh state
	assert.Empty(t, clone.fields)
	assert.Empty(t, clone.pendingFilter)
	assert.Empty(t, clone.joins)
	assert.Equal(t, uint64(0), clone.limit)
	assert.Equal(t, uint64(0), clone.offset)
	assert.Empty(t, clone.fromTable)

	// But should preserve field configs and placeholder
	assert.Equal(t, original.fieldConfigs, clone.fieldConfigs)
	assert.Equal(t, original.placeholder, clone.placeholder)
}

func TestToConditionEdgeCases(t *testing.T) {
	b := NewSQLBuilder()

	t.Run("LIKE with non-string value", func(t *testing.T) {
		// LIKE with non-string should fall back to default
		condition := b.toCondition(pendingFilter{
			FieldConfig: FieldConfig{DBField: "field", Operator: LIKE},
			Value:       123,
		})
		assert.NotNil(t, condition)
	})

	t.Run("ILIKE with non-string value", func(t *testing.T) {
		condition := b.toCondition(pendingFilter{
			FieldConfig: FieldConfig{DBField: "field", Operator: ILIKE},
			Value:       123,
		})
		assert.NotNil(t, condition)
	})

	t.Run("EXPR operator", func(t *testing.T) {
		condition := b.toCondition(pendingFilter{
			FieldConfig: FieldConfig{DBField: "field", Operator: EXPR},
			Value:       "LIKE ?",
			Args:        []any{"%test%"},
		})
		assert.NotNil(t, condition)
	})

	t.Run("EXPR_EQ operator", func(t *testing.T) {
		condition := b.toCondition(pendingFilter{
			FieldConfig: FieldConfig{DBField: "field", Operator: EXPR_EQ},
			Value:       "NOW()",
		})
		assert.NotNil(t, condition)
	})

	t.Run("unknown operator returns default", func(t *testing.T) {
		condition := b.toCondition(pendingFilter{
			FieldConfig: FieldConfig{DBField: "field", Operator: "unknown"},
			Value:       "something",
		})
		assert.NotNil(t, condition) // returns default condition "1=1"
	})
}

// ==========================================================================
// ==========================================================================
// ==========================================================================
// ==========================================================================
// ==========================================================================
// ==========================================================================
// ==========================================================================
// ==========================================================================
// ==========================================================================
// ==========================================================================
// ==========================================================================

func TestApplySort(t *testing.T) {
	t.Run("apply sort with field mapping", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithFieldConfig("user_id", "users.id", EQ)

		b.ApplySort("user_id", "ASC")

		assert.Equal(t, "id", b.sort.Field) // после cleanSortField
		assert.Equal(t, "ASC", b.sort.Order)
		assert.NoError(t, b.err)
	})

	t.Run("apply sort with function in field config", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithFieldConfig("name", "LOWER(users.name)", LIKE)

		b.ApplySort("name", "DESC")

		assert.Equal(t, "name", b.sort.Field) // очистилось от LOWER() и алиаса
		assert.Equal(t, "DESC", b.sort.Order)
	})

	t.Run("apply sort with empty field does nothing", func(t *testing.T) {
		b := NewSQLBuilder().WithFrom("users")
		b.ApplySort("", "ASC")

		assert.Empty(t, b.sort.Field)
		assert.Empty(t, b.sort.Order)
	})

	t.Run("apply sort with unknown field uses field as is", func(t *testing.T) {
		b := NewSQLBuilder().WithFrom("users")
		b.ApplySort("unknown_field", "ASC")

		assert.Equal(t, "unknown_field", b.sort.Field)
	})

	t.Run("apply sort with invalid order won't sets error", func(t *testing.T) {
		b := NewSQLBuilder().WithFrom("users").WithField("*")
		b.ApplySort("id", "INVALID")

		sql, _, err := b.BuildSelect()

		assert.NoError(t, err)
		// Invalid SQL possible, but that's the way it is
		assert.Contains(t, sql, "ORDER BY id INVALID")
	})
}

func TestCleanSortField(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		expectErr bool
	}{
		{
			name:     "simple field",
			input:    "id",
			expected: "id",
		},
		{
			name:     "field with table alias",
			input:    "users.id",
			expected: "id",
		},
		{
			name:     "field with schema and table",
			input:    "public.users.id",
			expected: "id",
		},
		{
			name:     "function with one argument",
			input:    "LOWER(name)",
			expected: "name",
		},
		{
			name:     "function with table alias",
			input:    "LOWER(users.name)",
			expected: "name",
		},
		{
			name:      "function with multiple arguments",
			input:     "SUBSTRING(name, 1, 5)",
			expected:  "",
			expectErr: true,
		},
		{
			name:      "nested functions",
			input:     "LOWER(SUBSTRING(name, 1, 5))",
			expected:  "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewSQLBuilder()
			b.sort = SortConfig{Field: tt.input, Order: "ASC"}
			b.cleanSortField()

			if tt.expectErr {
				assert.NotNil(t, b.err)
			} else {
				assert.NoError(t, b.err)
				assert.Equal(t, tt.expected, b.sort.Field)
			}
		})
	}
}

func TestApplyFilterWithMultipleCTEs(t *testing.T) {
	t.Run("filter applies to first matching CTE", func(t *testing.T) {
		main := NewSQLBuilder().WithFrom("final").WithField("*")

		cte1 := main.WithCTE("cte1", "c1")
		cte1.WithFrom("table1").WithField("id, name")

		cte2 := main.WithCTE("cte2", "c2")
		cte2.WithFrom("table2").WithField("id, status")

		main.WithFieldConfig("status", "status", EQ)
		main.ApplyFilter("status", "active")

		assert.Empty(t, cte1.pendingFilter)
		// Фильтр должен попасть в cte2, потому что поле status есть в cte2.fields
		assert.Len(t, cte2.pendingFilter, 1)
		assert.Empty(t, main.pendingFilter)
	})

	t.Run("filter applies to main when not in any CTE", func(t *testing.T) {
		main := NewSQLBuilder().WithFrom("final").WithField("*")

		cte1 := main.WithCTE("cte1", "c1")
		cte1.WithFrom("table1").WithField("id")

		main.WithFieldConfig("global_status", "status", EQ)
		main.ApplyFilter("global_status", "active")

		assert.Len(t, main.pendingFilter, 1)
		assert.Empty(t, cte1.pendingFilter)
	})
}

func TestBuildCountWithCTE(t *testing.T) {
	t.Run("count with CTE that has filter", func(t *testing.T) {
		main := NewSQLBuilder().WithFrom("active_users").WithField("*")

		cte := main.WithCTE("active_users", "au")
		cte.WithFrom("users").WithField("id").
			Where(EQ, "active", true)

		sql, args, err := main.BuildCount()

		require.NoError(t, err)
		assert.Contains(t, sql, "WITH active_users AS (")
		assert.Contains(t, sql, "WHERE (active = $1)")
		assert.Contains(t, sql, "SELECT COUNT(*) FROM (")
		assert.Len(t, args, 1)
	})
}

func TestSortIfValidation(t *testing.T) {
	t.Run("sort if with valid order", func(t *testing.T) {
		b := NewSQLBuilder().WithFrom("users")
		b.SortIf("id", "DESC")
		assert.Equal(t, "id", b.sort.Field)
		assert.Equal(t, "DESC", b.sort.Order)
	})

	t.Run("sort if with empty field does nothing", func(t *testing.T) {
		b := NewSQLBuilder().WithFrom("users")
		b.Sort("default", "ASC")
		b.SortIf("", "DESC")
		assert.Equal(t, "default", b.sort.Field) // не изменилось
	})
}

func TestSortAndApplySortConflict(t *testing.T) {
	t.Run("ApplySort overrides previous Sort", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithFieldConfig("user_id", "users.id", EQ)

		// Сначала жёсткая сортировка
		b.Sort("created_at", "DESC")
		assert.Equal(t, "created_at", b.sort.Field)
		assert.Equal(t, "DESC", b.sort.Order)

		// Потом динамическая сортировка из запроса
		b.ApplySort("user_id", "ASC")

		// Должна переопределиться
		assert.Equal(t, "id", b.sort.Field) // после cleanSortField
		assert.Equal(t, "ASC", b.sort.Order)
	})

	t.Run("Sort overrides previous ApplySort", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithFieldConfig("user_id", "users.id", EQ)

		// Сначала динамическая сортировка
		b.ApplySort("user_id", "ASC")
		assert.Equal(t, "id", b.sort.Field)

		// Потом жёсткая сортировка (например, сортировка по умолчанию)
		b.Sort("name", "DESC")

		// Должна переопределиться
		assert.Equal(t, "name", b.sort.Field)
		assert.Equal(t, "DESC", b.sort.Order)
	})

	t.Run("SortIf with empty field does not override ApplySort", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithFieldConfig("user_id", "users.id", EQ)

		// Применили динамическую сортировку
		b.ApplySort("user_id", "ASC")

		// Пытаемся применить SortIf с пустым полем
		b.SortIf("", "DESC")

		// Сортировка не должна измениться
		assert.Equal(t, "id", b.sort.Field)
		assert.Equal(t, "ASC", b.sort.Order)
	})

	t.Run("chain Sort and ApplySort with same field", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("users").
			WithFieldConfig("user_name", "LOWER(users.name)", LIKE)

		// Жёсткая сортировка
		b.Sort("id", "ASC")

		// Динамическая сортировка по другому полю
		b.ApplySort("user_name", "DESC")

		assert.Equal(t, "name", b.sort.Field)
		assert.Equal(t, "DESC", b.sort.Order)
	})

	t.Run("Apply Sort with Sort in CTE", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFieldConfig("user_name", "LOWER(users.name)", LIKE)

		cte := b.WithCTE("cte", "c").WithField("*").WithFrom("users u")
		cte.Sort("u.id", "desc")

		b.WithFrom("cte c").WithField("*")

		// Динамическая сортировка по другому полю
		b.ApplySort("user_name", "DESC")
		sql, _, err := b.BuildSelect()

		assert.NoError(t, err)
		assert.Equal(t, "name", b.sort.Field)
		assert.Equal(t, "DESC", b.sort.Order)
		assert.Contains(t, sql, "ORDER BY u.id desc")
		assert.Contains(t, sql, "ORDER BY name DESC")
	})
}

func TestGroupBy(t *testing.T) {
	t.Run("simple group by", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("my_table mt").
			WithField("id, max(updated_at) as last_updated").
			WithGroup("mt.id")

		sql, args, err := b.BuildSelect()

		assert.NoError(t, err)
		assert.Contains(t, sql, "GROUP BY mt.id")
		assert.Len(t, args, 0)
	})

	t.Run("simple group by with WHERE", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("my_table mt").
			WithField("id, max(updated_at) as last_updated").
			WithGroup("mt.id").
			Where(EQ, "mt.id", 13)

		sql, args, err := b.BuildSelect()

		assert.NoError(t, err)
		assert.Contains(t, sql, "GROUP BY mt.id")
		assert.Contains(t, sql, "WHERE (mt.id = $1)")
		assert.Len(t, args, 1)
		assert.Contains(t, args, 13)
	})

	t.Run("group by with multiple fields", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("orders o").
			WithField("o.user_id, o.status, COUNT(*) as total").
			WithGroup("o.user_id, o.status").
			Sort("total", "DESC")

		sql, args, err := b.BuildSelect()

		assert.NoError(t, err)
		assert.Contains(t, sql, "GROUP BY o.user_id, o.status")
		assert.Contains(t, sql, "ORDER BY total DESC")
		assert.Empty(t, args)
	})

	t.Run("group by in CTE", func(t *testing.T) {
		b := NewSQLBuilder()

		cte := b.WithCTE("preload", "prd").
			WithFrom("my_table mt").
			WithField("id, max(updated_at) as last_updated").
			WithGroup("mt.id").Where(EQ, "mt.id", 22)

		sql, args, err := cte.buildCTEBaseSelect().ToSql()

		assert.NoError(t, err)
		assert.Contains(t, sql, "GROUP BY mt.id")
		assert.Len(t, args, 1)
		assert.Contains(t, args, 22)
	})

	t.Run("group by with empty string", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("my_table mt").
			WithField("id, count(*)").
			WithGroup("")

		sql, _, err := b.BuildSelect()

		// return SQL without GROUP BY
		assert.NoError(t, err)
		assert.NotContains(t, sql, "GROUP BY")
	})

	t.Run("group by without fields in SELECT", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("my_table mt").
			WithGroup("mt.id")

		sql, _, err := b.BuildSelect()

		assert.ErrorContains(t, err, "select statements must have at least one result column")
		assert.Empty(t, sql)
	})

	t.Run("group by with expression", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("orders").
			WithField("date_trunc('month', created_at) as month, count(*)").
			WithGroup("date_trunc('month', created_at)")

		sql, _, err := b.BuildSelect()

		assert.NoError(t, err)
		assert.Contains(t, sql, "GROUP BY date_trunc('month', created_at)")
	})

	t.Run("group by with invalid field", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFrom("my_table mt").
			WithField("id, count(*)").
			WithGroup("non_existent_field")

		sql, _, err := b.BuildSelect()

		// Squirrel ошибку не вернёт — это ошибка выполнения
		assert.NoError(t, err)
		assert.Contains(t, sql, "GROUP BY non_existent_field")
		// Ошибка придёт от PostgreSQL: column "non_existent_field" does not exist
	})
}

func TestApplyFilterBehavior(t *testing.T) {
	t.Run("simple query", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFieldConfig("name", "name", EQ)

		b.WithFrom("my_table").
			WithField("id, max(updated_at) as last_updated, surname, name, full_name")

		filter := map[string]string{
			"search_name": "Tom",
		}

		for key, val := range filter {
			after, found := strings.CutPrefix(key, "search_")
			if !found {
				continue
			}
			b.ApplyFilter(after, val)
		}

		sql, args, err := b.BuildSelect()

		assert.NoError(t, err)
		assert.Contains(t, sql, "WHERE (name = $1)")
		assert.Len(t, args, 1)
	})

	t.Run("simple query with error", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFieldConfig("name", "name", EQ)

		b.WithFrom("my_table").
			WithField("id, max(updated_at) as last_updated, surname, name, full_name")

		filter := map[string]string{
			"search_something": "sample_thing",
		}

		for key, val := range filter {
			after, found := strings.CutPrefix(key, "search_")
			if !found {
				continue
			}
			b.ApplyFilter(after, val)
		}

		sql, args, err := b.BuildSelect()

		// in fieldConfigs has no fields like `something`
		assert.ErrorContains(t, err, "не найдено в настройках полей запроса")
		assert.NotContains(t, sql, "WHERE (something = $1)")
		assert.Len(t, args, 0)
	})

	t.Run("query with multiple cte [ NO ALIASES ]", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFieldConfig("name", "name", EQ) // WITHOUT ALIAS!

		cte1 := b.WithCTE("preload_data", "prd").
			WithFrom("my_table").
			WithField("id, max(updated_at) as last_updated, surname, name, full_name") // FIRST MATCHING = apply filter

		cte2 := b.WithCTE("final_query", "fq").
			WithFrom("preload_data prd").
			WithField("id, last_updated, surname, name, full_name")
		b.WithFrom("final_query fq").WithField("*")

		filter := map[string]string{
			"search_name": "Tom",
		}

		for key, val := range filter {
			after, found := strings.CutPrefix(key, "search_")
			if !found {
				continue
			}
			b.ApplyFilter(after, val)
		}

		sql, args, err := cte1.buildCTEBaseSelect().PlaceholderFormat(squirrel.Dollar).ToSql()

		assert.NoError(t, err)
		assert.Contains(t, sql, "WHERE (name = $1)")
		assert.Len(t, args, 1)

		sql, args, err = cte2.buildCTEBaseSelect().ToSql()

		assert.NoError(t, err)
		assert.NotContains(t, sql, "WHERE (")
		assert.Len(t, args, 0)

		sql, args, err = b.BuildSelect()

		assert.NoError(t, err)
		// there is no WHERE condition in main-sql
		assert.Contains(t, sql, "FROM final_query fq LIMIT 7")
		assert.Len(t, args, 1)
	})

	t.Run("query with multiple cte [ fieldConfigs HAS ALIAS ]", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFieldConfig("name", "prd.name", EQ) // WITH ALIAS!

		cte1 := b.WithCTE("preload_data", "prd").WithFrom("my_table").
			WithField("id, max(updated_at) as last_updated, surname, name, full_name") // FIRST MATCHING = apply filter

		cte2 := b.WithCTE("final_query", "fq").WithFrom("preload_data pd").
			WithField("prd.id, prd.last_updated, prd.surname, prd.name, prd.full_name")

		b.WithFrom("final_query fq").WithField("*")

		filter := map[string]string{
			"search_name": "Tom",
		}

		for key, val := range filter {
			after, found := strings.CutPrefix(key, "search_")
			if !found {
				continue
			}
			b.ApplyFilter(after, val)
		}

		sql, args, err := cte1.buildCTEBaseSelect().PlaceholderFormat(squirrel.Dollar).ToSql()

		assert.NoError(t, err)
		assert.Contains(t, sql, "WHERE (prd.name = $1)")
		assert.Len(t, args, 1)

		sql, args, err = cte2.buildCTEBaseSelect().PlaceholderFormat(Dollar).ToSql()

		assert.NoError(t, err)
		assert.NotContains(t, sql, "WHERE (")
		assert.Len(t, args, 0)

		sql, args, err = b.BuildSelect()

		assert.NoError(t, err)
		// there is no WHERE condition in main-sql
		assert.Contains(t, sql, "FROM final_query fq LIMIT 7")
		assert.Len(t, args, 1)
	})

	t.Run("query with multiple cte [ everyone HAS ALIAS ]", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFieldConfig("name", "prd.name", EQ) // WITH ALIAS!

		cte1 := b.WithCTE("preload_data", "prd").
			WithFrom("my_table mt").                                                               // WITH ALIAS!
			WithField("mt.id, max(updated_at) as last_updated, mt.surname, mt.name, mt.full_name") // WITH ALIAS!

		cte2 := b.WithCTE("final_query", "fq").
			WithFrom("preload_data prd").                                               // WITH ALIAS!                                          // WITH ALIAS!
			WithField("prd.id, prd.last_updated, prd.surname, prd.name, prd.full_name") // HERE's MATCHING = apply filter

		b.WithFrom("final_query fq").WithField("*")

		filter := map[string]string{
			"search_name": "Tom",
		}

		for key, val := range filter {
			after, found := strings.CutPrefix(key, "search_")
			if !found {
				continue
			}
			b.ApplyFilter(after, val)
		}

		sql, args, err := cte1.buildCTEBaseSelect().PlaceholderFormat(Dollar).ToSql()

		assert.NoError(t, err)
		assert.NotContains(t, sql, "WHERE (")
		assert.Len(t, args, 0)

		sql, args, err = cte2.buildCTEBaseSelect().PlaceholderFormat(Dollar).ToSql()

		assert.NoError(t, err)
		assert.Contains(t, sql, "WHERE (prd.name = $1)")
		assert.Len(t, args, 1)

		sql, args, err = b.BuildSelect()

		assert.NoError(t, err)
		// there is no WHERE condition in main-sql
		assert.Contains(t, sql, "FROM final_query fq LIMIT 7")
		assert.Len(t, args, 1)
	})

	t.Run("query with expression into fieldconfig [ HAS ALIAS, NO FIELDS]", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFieldConfig("snils", "persons.snils2bcd64(r.snils)", EXPR_EQ)

		cte1 := b.WithCTE("preload_data", "prd").
			WithFrom("my_table mt").
			WithField("mt.id, mt.surname, mt.name")

		cte2 := b.WithCTE("final_query", "fq").
			WithFrom("preload_data prd").
			WithField("prd.id, prd.last_updated, prd.surname, prd.name, prd.full_name")

		b.WithFrom("final_query fq").
			WithField("fq.name, fq.snils, fq.last_updated") // no matching, apply filter

		filter := map[string]string{
			"search_snils": "123-456-789 33",
		}

		for key, val := range filter {
			after, found := strings.CutPrefix(key, "search_")
			if !found {
				continue
			}

			if after == "snils" {
				b.ApplyFilter(after, "persons.snils2bcd64(?)", val)
				continue
			}

			b.ApplyFilter(after, val)
		}

		sql, args, err := cte1.buildCTEBaseSelect().PlaceholderFormat(Dollar).ToSql()

		assert.NoError(t, err)
		assert.NotContains(t, sql, "WHERE (")
		assert.Len(t, args, 0)

		sql, args, err = cte2.buildCTEBaseSelect().PlaceholderFormat(Dollar).ToSql()

		assert.NoError(t, err)
		assert.NotContains(t, sql, "WHERE (")
		assert.Len(t, args, 0)

		sql, args, err = b.BuildSelect()

		assert.NoError(t, err)
		// there is no WHERE condition in main-sql
		assert.Contains(t, sql, "FROM final_query fq WHERE (persons.snils2bcd64(r.snils) = persons.snils2bcd64($1)) LIMIT 7")
		assert.Len(t, args, 1)
	})

	t.Run("query with expression into fieldconfig [ HAS ALIAS, HAS FIELD]", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFieldConfig("snils", "persons.snils2bcd64(r.snils)", EXPR_EQ)

		cte1 := b.WithCTE("preload_data", "prd").
			WithFrom("my_table mt").
			WithField("mt.id, mt.surname, mt.name")

		cte2 := b.WithCTE("final_query", "fq").
			WithFrom("preload_data prd").
			WithField("prd.id, prd.last_updated, prd.surname, prd.name, prd.full_name, r.snils"). // HERE's MATCHING, apply filter
			WithInnerJoin("persons.recipients r", "r.id = prd.id")

		b.WithFrom("final_query fq").
			WithField("fq.name, fq.snils, fq.last_updated")

		filter := map[string]string{
			"search_snils": "123-456-789 33",
		}

		for key, val := range filter {
			after, found := strings.CutPrefix(key, "search_")
			if !found {
				continue
			}

			if after == "snils" {
				b.ApplyFilter(after, "persons.snils2bcd64(?)", val)
				continue
			}

			b.ApplyFilter(after, val)
		}

		sql, args, err := cte1.buildCTEBaseSelect().PlaceholderFormat(Dollar).ToSql()

		assert.NoError(t, err)
		assert.NotContains(t, sql, "WHERE (")
		assert.Len(t, args, 0)

		sql, args, err = cte2.buildCTEBaseSelect().PlaceholderFormat(Dollar).ToSql()

		assert.NoError(t, err)
		assert.Contains(t, sql, "WHERE (persons.snils2bcd64(r.snils) = persons.snils2bcd64($1))")
		assert.Len(t, args, 1)

		sql, args, err = b.BuildSelect()

		assert.NoError(t, err)
		// there is no WHERE condition in main-sql
		assert.Contains(t, sql, "FROM final_query fq LIMIT 7")
		assert.Len(t, args, 1)
	})

	t.Run("field config matches multiple field patterns in query [ WRONG MATCHING ]", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFieldConfig("user_id", "u.user_id", EQ)

		cte1 := b.WithCTE("preload_data", "prd").
			WithFrom("my_table mt").
			WithField("id, surname, name, full_name") // ID WITHOUT ALIAS, MATCHING IS HERE: apply filter

		cte2 := b.WithCTE("final_query", "fq").
			WithFrom("preload_data pd").
			WithField("prd.id, prd.surname, prd.name, prd.full_name, user_id"). // user_id WITHOUT alias
			WithInnerJoin("admin.users u", "u.id = prd.id")

		b.WithFrom("final_query fq").WithField("fq.name, fq.snils, fq.last_updated, user_id")

		filter := map[string]string{
			"search_user_id": "344",
		}

		/* Here ApplyFilter add WHERE condition in main query, because not find contains field */
		for key, val := range filter {
			after, found := strings.CutPrefix(key, "search_")
			if !found {
				continue
			}
			b.ApplyFilter(after, val)
		}

		sql, args, err := cte1.buildCTEBaseSelect().PlaceholderFormat(Dollar).ToSql()

		assert.NoError(t, err)
		assert.Contains(t, sql, "WHERE (u.user_id = $1)")
		assert.Len(t, args, 1)

		sql, args, err = cte2.buildCTEBaseSelect().PlaceholderFormat(Dollar).ToSql()

		assert.NoError(t, err)
		assert.NotContains(t, sql, "WHERE (")
		assert.Len(t, args, 0)

		sql, args, err = b.BuildSelect()

		assert.NoError(t, err)
		// there is no WHERE condition in main-sql
		assert.Contains(t, sql, "FROM final_query fq LIMIT 7")
		assert.Len(t, args, 1)
	})

	t.Run("field config matches multiple field patterns in query [ RIGHT MATCHING ]", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFieldConfig("user_id", "fq.user_id", EQ)

		cte1 := b.WithCTE("preload_data", "prd").
			WithFrom("my_table mt").
			WithField("mt.id, mt.surname, mt.name, mt.full_name") // with alias, no matching

		cte2 := b.WithCTE("final_query", "fq").
			WithFrom("preload_data pd").
			WithField("prd.id, prd.surname, prd.name, prd.full_name, u.user_id"). // with alias, no matching
			WithInnerJoin("admin.users u", "u.id = prd.id")

		b.WithFrom("final_query fq").WithField("fq.name, fq.snils, fq.last_updated, fq.user_id") // here's matching: apply filter

		filter := map[string]string{
			"search_user_id": "344",
		}

		/* Here ApplyFilter add WHERE condition in main query, because not find contains field */
		for key, val := range filter {
			after, found := strings.CutPrefix(key, "search_")
			if !found {
				continue
			}
			b.ApplyFilter(after, val)
		}

		sql, args, err := cte1.buildCTEBaseSelect().PlaceholderFormat(Dollar).ToSql()

		assert.NoError(t, err)
		assert.NotContains(t, sql, "WHERE (")
		assert.Len(t, args, 0)

		sql, args, err = cte2.buildCTEBaseSelect().PlaceholderFormat(Dollar).ToSql()

		assert.NoError(t, err)
		assert.NotContains(t, sql, "WHERE (")
		assert.Len(t, args, 0)

		sql, args, err = b.BuildSelect()

		assert.NoError(t, err)
		// there is no WHERE condition in main-sql
		assert.Contains(t, sql, "FROM final_query fq WHERE (fq.user_id = $1) LIMIT 7")
		assert.Len(t, args, 1)
	})

	t.Run("field config matches multiple field patterns in query [ CASE WHEN mathcing alias in AS position ]", func(t *testing.T) {
		b := NewSQLBuilder().
			WithFieldConfig("user_id", "fq.user_id", EQ)

		cte1 := b.WithCTE("preload_data", "prd").
			WithFrom("my_table mt").
			WithField(`
			mt.id, mt.surname, mt.name, mt.full_name
			, case when 1 < 2 then 88 else 99 end as user_id
		`) // with alias, no matching

		cte2 := b.WithCTE("final_query", "fq").
			WithFrom("preload_data pd").
			WithField("prd.id, prd.surname, prd.name, prd.full_name, u.user_id"). // with alias, no matching
			WithInnerJoin("admin.users u", "u.id = prd.id")

		b.WithFrom("final_query fq").WithField("*") // here's matching: apply filter

		filter := map[string]string{
			"search_user_id": "344",
		}

		/* Here ApplyFilter add WHERE condition in main query, because not find contains field */
		for key, val := range filter {
			after, found := strings.CutPrefix(key, "search_")
			if !found {
				continue
			}
			b.ApplyFilter(after, val)
		}

		sql, args, err := cte1.buildCTEBaseSelect().PlaceholderFormat(Dollar).ToSql()

		assert.NoError(t, err)
		assert.NotContains(t, sql, "WHERE (")
		assert.Len(t, args, 0)

		sql, args, err = cte2.buildCTEBaseSelect().PlaceholderFormat(Dollar).ToSql()

		assert.NoError(t, err)
		assert.NotContains(t, sql, "WHERE (")
		assert.Len(t, args, 0)

		sql, args, err = b.BuildSelect()

		assert.NoError(t, err)
		// there is no WHERE condition in main-sql
		assert.Contains(t, sql, "FROM final_query fq WHERE (fq.user_id = $1) LIMIT 7")
		assert.Len(t, args, 1)
	})
}
