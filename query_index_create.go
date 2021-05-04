package bun

import "github.com/uptrace/bun/sqlfmt"

type CreateIndexQuery struct {
	whereBaseQuery

	unique       bool
	fulltext     bool
	spatial      bool
	concurrently bool
	ifNotExists  bool

	index   sqlfmt.QueryWithArgs
	using   sqlfmt.QueryWithArgs
	include []sqlfmt.QueryWithArgs
}

func NewCreateIndexQuery(db *DB) *CreateIndexQuery {
	q := &CreateIndexQuery{
		whereBaseQuery: whereBaseQuery{
			baseQuery: baseQuery{
				db:  db,
				dbi: db.DB,
			},
		},
	}
	return q
}

func (q *CreateIndexQuery) DB(db DBI) *CreateIndexQuery {
	q.dbi = db
	return q
}

func (q *CreateIndexQuery) Model(model interface{}) *CreateIndexQuery {
	q.setTableModel(model)
	return q
}

func (q *CreateIndexQuery) Unique() *CreateIndexQuery {
	q.unique = true
	return q
}

func (q *CreateIndexQuery) Concurrently() *CreateIndexQuery {
	q.concurrently = true
	return q
}

func (q *CreateIndexQuery) IfNotExists() *CreateIndexQuery {
	q.ifNotExists = true
	return q
}

//------------------------------------------------------------------------------

func (q *CreateIndexQuery) Index(query string, args ...interface{}) *CreateIndexQuery {
	q.index = sqlfmt.SafeQuery(query, args)
	return q
}

//------------------------------------------------------------------------------

func (q *CreateIndexQuery) Table(tables ...string) *CreateIndexQuery {
	for _, table := range tables {
		q.addTable(sqlfmt.UnsafeIdent(table))
	}
	return q
}

func (q *CreateIndexQuery) TableExpr(query string, args ...interface{}) *CreateIndexQuery {
	q.addTable(sqlfmt.SafeQuery(query, args))
	return q
}

func (q *CreateIndexQuery) ModelTableExpr(query string, args ...interface{}) *CreateIndexQuery {
	q.modelTable = sqlfmt.SafeQuery(query, args)
	return q
}

func (q *CreateIndexQuery) Using(query string, args ...interface{}) *CreateIndexQuery {
	q.using = sqlfmt.SafeQuery(query, args)
	return q
}

//------------------------------------------------------------------------------

func (q *CreateIndexQuery) Column(columns ...string) *CreateIndexQuery {
	for _, column := range columns {
		q.addColumn(sqlfmt.UnsafeIdent(column))
	}
	return q
}

func (q *CreateIndexQuery) ColumnExpr(query string, args ...interface{}) *CreateIndexQuery {
	q.addColumn(sqlfmt.SafeQuery(query, args))
	return q
}

func (q *CreateIndexQuery) ExcludeColumn(columns ...string) *CreateIndexQuery {
	q.excludeColumn(columns)
	return q
}

//------------------------------------------------------------------------------

func (q *CreateIndexQuery) Include(columns ...string) *CreateIndexQuery {
	for _, column := range columns {
		q.include = append(q.include, sqlfmt.UnsafeIdent(column))
	}
	return q
}

func (q *CreateIndexQuery) IncludeExpr(query string, args ...interface{}) *CreateIndexQuery {
	q.include = append(q.include, sqlfmt.SafeQuery(query, args))
	return q
}

//------------------------------------------------------------------------------

func (q *CreateIndexQuery) Where(query string, args ...interface{}) *CreateIndexQuery {
	q.addWhere(sqlfmt.SafeQueryWithSep(query, args, " AND "))
	return q
}

func (q *CreateIndexQuery) WhereOr(query string, args ...interface{}) *CreateIndexQuery {
	q.addWhere(sqlfmt.SafeQueryWithSep(query, args, " OR "))
	return q
}

func (q *CreateIndexQuery) WhereGroup(sep string, fn func(*WhereQuery)) *CreateIndexQuery {
	q.addWhereGroup(sep, fn)
	return q
}

//------------------------------------------------------------------------------

func (q *CreateIndexQuery) AppendQuery(fmter sqlfmt.QueryFormatter, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}

	b = append(b, "CREATE "...)

	if q.unique {
		b = append(b, "UNIQUE "...)
	}
	if q.fulltext {
		b = append(b, "FULLTEXT "...)
	}
	if q.spatial {
		b = append(b, "SPATIAL "...)
	}

	b = append(b, "INDEX "...)

	if q.concurrently {
		b = append(b, "CONCURRENTLY "...)
	}
	if q.ifNotExists {
		b = append(b, "IF NOT EXISTS "...)
	}

	b, err = q.index.AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}

	b = append(b, " ON "...)
	b, err = q.appendFirstTable(fmter, b)
	if err != nil {
		return nil, err
	}

	if !q.using.IsZero() {
		b = append(b, " USING "...)
		b, err = q.using.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	b = append(b, " ("...)
	for i, col := range q.columns {
		if i > 0 {
			b = append(b, ", "...)
		}
		b, err = col.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}
	}
	b = append(b, ')')

	if len(q.include) > 0 {
		b = append(b, " INCLUDE ("...)
		for i, col := range q.include {
			if i > 0 {
				b = append(b, ", "...)
			}
			b, err = col.AppendQuery(fmter, b)
			if err != nil {
				return nil, err
			}
		}
		b = append(b, ')')
	}

	b, err = q.appendWhere(fmter, b)
	if err != nil {
		return nil, err
	}

	return b, nil
}