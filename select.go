package builder

import (
	"fmt"
)

type SelectQuery struct {
	from    []*Table
	columns []Column
	joins   []*join
	where   Where
	order   []Order
	limit   string
	offset  string
	group   []Group
	binds   map[string]any
	isSub   bool
}

func NewSelect() *SelectQuery {
	return &SelectQuery{
		binds: make(map[string]any),
	}
}

func (q *SelectQuery) checkTable(table *Table) bool {
	if q.isSub {
		return true
	}

	for _, t := range q.from {
		if t == table {
			return true
		}
	}

	for _, j := range q.joins {
		if j.Table == table {
			j.Used = true

			return true
		}
	}

	return false
}

func (q *SelectQuery) addBind(key string, value any) {
	q.binds[key] = value
}

func (q *SelectQuery) From(t ...*Table) *SelectQuery {
	q.from = append(q.from, t...)

	return q
}

func (q *SelectQuery) Column(c ...Column) *SelectQuery {
	q.columns = append(q.columns, c...)

	return q
}

func (q *SelectQuery) LeftJoin(table *Table, on On) *SelectQuery {
	q.joins = append(q.joins, &join{
		Table: table,
		On:    on,
		Left:  true,
	})

	return q
}

func (q *SelectQuery) Where(w Where) *SelectQuery {
	q.where = w

	return q
}

func (q *SelectQuery) Order(o ...Order) *SelectQuery {
	q.order = append(q.order, o...)

	return q
}

func (q *SelectQuery) Limit(limit int) *SelectQuery {
	if limit <= 0 {
		return q
	}

	q.limit = "limit_" + randStr()
	q.addBind(q.limit, limit)

	return q
}

func (q *SelectQuery) Offset(offset int) *SelectQuery {
	if offset <= 0 {
		return q
	}

	q.offset = "offset_" + randStr()
	q.addBind(q.offset, offset)

	return q
}

func (q *SelectQuery) Group(g ...Group) *SelectQuery {
	q.group = append(q.group, g...)

	return q
}

func (q *SelectQuery) IsSub() *SelectQuery {
	q.isSub = true

	return q
}

func (q *SelectQuery) getSelect() (string, error) {
	if len(q.columns) == 0 {
		return "", fmt.Errorf("no columns defined for select")
	}

	s := "SELECT "

	for i, col := range q.columns {
		c, err := col.gen(q)
		if err != nil {
			return "", err
		}

		s += c

		if i != len(q.columns)-1 {
			s += ", "
		}
	}

	return s, nil
}

func (q *SelectQuery) getFrom() (string, error) {
	if len(q.from) == 0 {
		return "", fmt.Errorf("no froms defined for select")
	}

	s := " FROM "

	for i, from := range q.from {
		sql, binds, err := from.gen()
		if err != nil {
			return "", err
		}

		for k, v := range binds {
			q.addBind(k, v)
		}

		s += sql

		if i != len(q.from)-1 {
			s += ", "
		}
	}

	return s, nil
}

func (q *SelectQuery) getWhere() (string, error) {
	if q.where == nil {
		return "", nil
	}

	where, binds, err := q.where.gen(q)
	if err != nil {
		return "", err
	}

	if where == "" {
		return "", nil
	}

	for k, v := range binds {
		q.addBind(k, v)
	}

	return " WHERE " + where, nil
}

func (q *SelectQuery) getOrder() (string, error) {
	if len(q.order) == 0 {
		return "", nil
	}

	s := " ORDER BY "

	for i, o := range q.order {
		if o.Table != nil {
			if !q.checkTable(o.Table) {
				return "", fmt.Errorf("table %s is not exist", o.Table)
			}

			s += o.Table.Alias + "."
		}

		s += o.Column

		if o.Desc {
			s += " DESC"
		}

		if i != len(q.order)-1 {
			s += ", "
		}
	}

	return s, nil
}

func (q *SelectQuery) getJoin() (string, error) {
	if len(q.joins) == 0 {
		return "", nil
	}

	s := ""

	for _, j := range q.joins {
		if !j.Used {
			continue
		}

		sj, err := j.Gen(q)
		if err != nil {
			return "", err
		}

		s += sj
	}

	return s, nil
}

func (q *SelectQuery) getGroup() (string, error) {
	if len(q.group) == 0 {
		return "", nil
	}

	s := " GROUP BY "

	for i, g := range q.group {
		sql, err := g.gen(q)
		if err != nil {
			return "", err
		}

		s += sql

		if i != len(q.group)-1 {
			s += ", "
		}
	}

	return s, nil
}

func (q *SelectQuery) Get() (string, map[string]any, error) {
	sel, err := q.getSelect()
	if err != nil {
		return "", nil, err
	}

	from, err := q.getFrom()
	if err != nil {
		return "", nil, err
	}

	where, err := q.getWhere()
	if err != nil {
		return "", nil, err
	}

	order, err := q.getOrder()
	if err != nil {
		return "", nil, err
	}

	j, err := q.getJoin()
	if err != nil {
		return "", nil, err
	}

	group, err := q.getGroup()
	if err != nil {
		return "", nil, err
	}

	limit := ""
	if q.limit != "" {
		limit = " LIMIT @" + q.limit
	}

	offset := ""
	if q.offset != "" {
		offset = " OFFSET @" + q.offset
	}

	return sel + from + j + where + group + order + limit + offset, q.binds, nil
}
