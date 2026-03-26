package sqlist

// // унифицируем сохранение фильтров, ожидающих обработки и добавления в финальный SQL
// // func (b *SQLBuilder) withPending(op Op, field string, value any, args ...any) pendingFilter {
// // 	return pendingFilter{
// // 		FieldConfig: FieldConfig{
// // 			DBField:  field,
// // 			Operator: op,
// // 		},
// // 		Value: value,
// // 		Args:  args,
// // 	}
// // }

// // // MapField возвращает настоящее имя колонки по псевдониму
// // func (b *SQLBuilder) mapField(alias string) string {
// // 	if cfg, ok := b.fieldConfigs[alias]; ok {
// // 		return cfg.DBField
// // 	}
// // 	return alias // если не нашли, возвращаем как есть
// // }

// // все ниже следующие методы - лишний мусор, т.к. обходимся WHere и FieldConfig-настройками

// // Eq добавляет условие равенства
// func (b *SQLBuilder) Eq(field string, value interface{}) *SQLBuilder {
// 	if value != nil {
// 		b.pendingFilter = append(b.pendingFilter,
// 			b.withPending(EQ, b.mapField(field), value),
// 		)
// 	}
// 	return b
// }

// // NotEq добавляет условие неравенства
// func (b *SQLBuilder) NotEq(field string, value interface{}) *SQLBuilder {
// 	if value != nil {
// 		b.pendingFilter = append(b.pendingFilter,
// 			b.withPending(NOT_EQ, b.mapField(field), value),
// 		)
// 	}
// 	return b
// }

// /*
//  * проблема в том, что мы не можем добавлять через данные оператор
//  * выражение!
//  * на выходе получаем, например, LIKE lower(?)%
//  */
// // Like добавляет условие LIKE
// func (b *SQLBuilder) Like(field string, value string) *SQLBuilder {
// 	if value != "" {
// 		b.pendingFilter = append(b.pendingFilter,
// 			b.withPending(LIKE, b.mapField(field), value+"%"),
// 		)
// 	}
// 	return b
// }

// // ILike добавляет условие ILIKE
// func (b *SQLBuilder) ILike(field string, value string) *SQLBuilder {
// 	if value != "" {
// 		b.pendingFilter = append(b.pendingFilter,
// 			b.withPending(ILIKE, b.mapField(field), "%"+value+"%"),
// 		)
// 	}
// 	return b
// }

// // In добавляет условие IN
// func (b *SQLBuilder) In(field string, values interface{}) *SQLBuilder {
// 	if values != nil {
// 		b.pendingFilter = append(b.pendingFilter,
// 			b.withPending(EQ, b.mapField(field), values),
// 		)
// 	}
// 	return b
// }

// // NotIn добавляет условие NOT IN
// func (b *SQLBuilder) NotIn(field string, values interface{}) *SQLBuilder {
// 	if values != nil {
// 		b.pendingFilter = append(b.pendingFilter,
// 			b.withPending(NOT_EQ, b.mapField(field), values),
// 		)
// 	}
// 	return b
// }

// // Between добавляет условие BETWEEN
// func (b *SQLBuilder) Between(field string, min, max interface{}) *SQLBuilder {
// 	if min != nil && max != nil {
// 		b.pendingFilter = append(b.pendingFilter,
// 			b.withPending(EXPR, b.mapField(field), b.mapField(field)+" BETWEEN ? AND ?", min, max),
// 		)
// 	}
// 	return b
// }

// // Gt добавляет условие "больше"
// func (b *SQLBuilder) Gt(field string, value interface{}) *SQLBuilder {
// 	if value != nil {
// 		b.pendingFilter = append(b.pendingFilter,
// 			b.withPending(GT, b.mapField(field), value),
// 		)
// 	}
// 	return b
// }

// // Lt добавляет условие "меньше"
// func (b *SQLBuilder) Lt(field string, value interface{}) *SQLBuilder {
// 	if value != nil {
// 		b.pendingFilter = append(b.pendingFilter,
// 			b.withPending(LT, b.mapField(field), value),
// 		)
// 	}
// 	return b
// }

// // Gte добавляет условие "больше или равно"
// func (b *SQLBuilder) Gte(field string, value interface{}) *SQLBuilder {
// 	if value != nil {
// 		b.pendingFilter = append(b.pendingFilter,
// 			b.withPending(GTE, b.mapField(field), value),
// 		)
// 	}
// 	return b
// }

// // Lte добавляет условие "меньше или равно"
// func (b *SQLBuilder) Lte(field string, value interface{}) *SQLBuilder {
// 	if value != nil {
// 		b.pendingFilter = append(b.pendingFilter,
// 			b.withPending(LTE, b.mapField(field), value),
// 		)
// 	}
// 	return b
// }

// // IsNull добавляет условие IS NULL
// func (b *SQLBuilder) IsNull(field string) *SQLBuilder {
// 	b.pendingFilter = append(b.pendingFilter,
// 		b.withPending(EQ, b.mapField(field), nil),
// 	)
// 	return b
// }

// // IsNotNull добавляет условие IS NOT NULL
// func (b *SQLBuilder) IsNotNull(field string) *SQLBuilder {
// 	b.pendingFilter = append(b.pendingFilter,
// 		b.withPending(NOT_EQ, b.mapField(field), nil),
// 	)
// 	return b
// }

// // Or группирует условия в OR
// // func (b *SQLBuilder) Or(conditions ...squirrel.Sqlizer) *SQLBuilder {
// // 	if len(conditions) > 0 {
// // 		b.whereConditions = append(b.whereConditions, squirrel.Or(conditions))
// // 	}
// // 	return b
// // }

// // // And группирует условия в AND (обычно не нужно)
// // func (b *SQLBuilder) And(conditions ...squirrel.Sqlizer) *SQLBuilder {
// // 	if len(conditions) > 0 {
// // 		b.whereConditions = append(b.whereConditions, squirrel.And(conditions))
// // 	}
// // 	return b
// // }

// func (b *SQLBuilder) Expr(field string, fullExpr string, args ...interface{}) *SQLBuilder {
// 	// как-то странно
// 	b.pendingFilter = append(b.pendingFilter,
// 		b.withPending(EXPR, field, fullExpr, nil),
// 	)

// 	return b
// }

// // ExprEq добавляет условие с функцией с правой стороны
// // Пример: persons.snils2bcd64(snils) = persons.snils2bcd64('111-111-111 11')
// func (b *SQLBuilder) ExprEq(leftField, rightExpr string, args ...interface{}) *SQLBuilder {
// 	fullExpr := leftField + " = " + rightExpr

// 	b.Expr(leftField, fullExpr, args...)

// 	return b
// }
