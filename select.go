package dbs

import (
	"bytes"
	"errors"
	"io"
	"strings"
)

const (
	k_SQL_CALC_FOUND_ROWS = "SQL_CALC_FOUND_ROWS"
	k_FOUND_ROWS          = "FOUND_ROWS()"
	k_COUNT               = "COUNT(1)"
)

type SelectBuilder struct {
	*builder
	*query
	*scan
	prefixes  statements
	options   statements
	columns   statements
	from      statements
	joins     statements
	wheres    statements
	groupBys  []string
	havings   statements
	orderBys  []string
	limit     Statement
	offset    Statement
	suffixes  statements
	foundRows bool
}

func (this *SelectBuilder) Clone() *SelectBuilder {
	var sb = NewSelectBuilder()
	sb.builder.d = this.builder.d
	sb.prefixes = this.prefixes
	sb.options = this.options
	sb.columns = this.columns
	sb.from = this.from
	sb.joins = this.joins
	sb.wheres = this.wheres
	sb.groupBys = this.groupBys
	sb.havings = this.havings
	sb.orderBys = this.orderBys
	sb.limit = this.limit
	sb.offset = this.offset
	sb.suffixes = this.suffixes
	sb.foundRows = this.foundRows
	return sb
}

func (this *SelectBuilder) Prefix(sql string, args ...interface{}) *SelectBuilder {
	this.prefixes = append(this.prefixes, NewStatement(sql, args...))
	return this
}

func (this *SelectBuilder) Options(options ...string) *SelectBuilder {
	for _, c := range options {
		if c == k_SQL_CALC_FOUND_ROWS {
			this.foundRows = true
		}
		this.options = append(this.options, NewStatement(c))
	}
	return this
}

func (this *SelectBuilder) Selects(columns ...string) *SelectBuilder {
	for _, c := range columns {
		this.columns = append(this.columns, NewStatement(c))
	}
	return this
}

func (this *SelectBuilder) Select(column interface{}, args ...interface{}) *SelectBuilder {
	var stmt = parseStmt(column, args...)

	if stmt != nil {
		this.columns = append(this.columns, stmt)
	}
	return this
}

func (this *SelectBuilder) From(table string, args ...string) *SelectBuilder {
	var ts []string
	ts = append(ts, this.quote(table))
	ts = append(ts, args...)
	this.from = append(this.from, NewStatement(strings.Join(ts, " ")))
	return this
}

func (this *SelectBuilder) Join(join, table, suffix string, args ...interface{}) *SelectBuilder {
	return this.join(join, table, suffix, args...)
}

func (this *SelectBuilder) RightJoin(table, suffix string, args ...interface{}) *SelectBuilder {
	return this.join("RIGHT JOIN", table, suffix, args...)
}

func (this *SelectBuilder) LeftJoin(table, suffix string, args ...interface{}) *SelectBuilder {
	return this.join("LEFT JOIN", table, suffix, args...)
}

func (this *SelectBuilder) join(join, table, suffix string, args ...interface{}) *SelectBuilder {
	var sql = []string{join, this.quote(table), suffix}
	this.joins = append(this.joins, NewStatement(strings.Join(sql, " "), args...))
	return this
}

func (this *SelectBuilder) Where(sql interface{}, args ...interface{}) *SelectBuilder {
	var stmt = parseStmt(sql, args...)
	if stmt != nil {
		this.wheres = append(this.wheres, stmt)
	}
	return this
}

func (this *SelectBuilder) GroupBy(groupBys ...string) *SelectBuilder {
	this.groupBys = append(this.groupBys, groupBys...)
	return this
}

func (this *SelectBuilder) Having(sql string, args ...interface{}) *SelectBuilder {
	this.havings = append(this.havings, NewStatement(sql, args...))
	return this
}

func (this *SelectBuilder) OrderBy(sql ...string) *SelectBuilder {
	this.orderBys = append(this.orderBys, sql...)
	return this
}

func (this *SelectBuilder) Limit(limit int64) *SelectBuilder {
	this.limit = NewStatement(" LIMIT ?", limit)
	return this
}

func (this *SelectBuilder) Offset(offset int64) *SelectBuilder {
	this.offset = NewStatement(" OFFSET ?", offset)
	return this
}

func (this *SelectBuilder) Suffix(sql string, args ...interface{}) *SelectBuilder {
	this.suffixes = append(this.suffixes, NewStatement(sql, args...))
	return this
}

func (this *SelectBuilder) ToSQL() (string, []interface{}, error) {
	var sqlBuffer = &bytes.Buffer{}
	var args = newArgs()
	if err := this.AppendToSQL(sqlBuffer, args); err != nil {
		return "", nil, err
	}
	sql, err := this.parseVal(sqlBuffer.String())
	if err != nil {
		return "", nil, err
	}
	return sql, args.values, nil
}

func (this *SelectBuilder) AppendToSQL(w io.Writer, args *Args) error {
	if len(this.columns) == 0 {
		return errors.New("SELECT statements must have at least on result column")
	}

	if len(this.prefixes) > 0 {
		if err := this.prefixes.AppendToSQL(w, " ", args); err != nil {
			return err
		}
		if _, err := io.WriteString(w, " "); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(w, "SELECT "); err != nil {
		return err
	}

	if len(this.options) > 0 {
		if err := this.options.AppendToSQL(w, " ", args); err != nil {
			return err
		}
		if _, err := io.WriteString(w, " "); err != nil {
			return err
		}
	}

	if len(this.columns) > 0 {
		if err := this.columns.AppendToSQL(w, ", ", args); err != nil {
			return err
		}
	}

	if len(this.from) > 0 {
		if _, err := io.WriteString(w, " FROM "); err != nil {
			return err
		}
		if err := this.from.AppendToSQL(w, ", ", args); err != nil {
			return err
		}
	}

	if len(this.joins) > 0 {
		if _, err := io.WriteString(w, " "); err != nil {
			return err
		}
		if err := this.joins.AppendToSQL(w, " ", args); err != nil {
			return err
		}
	}

	if len(this.wheres) > 0 {
		if _, err := io.WriteString(w, " WHERE "); err != nil {
			return err
		}
		if err := this.wheres.AppendToSQL(w, " AND ", args); err != nil {
			return err
		}
	}

	if len(this.groupBys) > 0 {
		if _, err := io.WriteString(w, " GROUP BY "); err != nil {
			return err
		}
		if _, err := io.WriteString(w, strings.Join(this.groupBys, ", ")); err != nil {
			return err
		}
	}

	if len(this.havings) > 0 {
		if _, err := io.WriteString(w, " HAVING "); err != nil {
			return err
		}
		if err := this.havings.AppendToSQL(w, " AND ", args); err != nil {
			return err
		}
	}

	if len(this.orderBys) > 0 {
		if _, err := io.WriteString(w, " ORDER BY "); err != nil {
			return err
		}
		if _, err := io.WriteString(w, strings.Join(this.orderBys, ", ")); err != nil {
			return err
		}
	}

	if this.limit != nil {
		if err := this.limit.AppendToSQL(w, args); err != nil {
			return err
		}
	}

	if this.offset != nil {
		if err := this.offset.AppendToSQL(w, args); err != nil {
			return err
		}
	}

	if len(this.suffixes) > 0 {
		if _, err := io.WriteString(w, " "); err != nil {
			return err
		}
		if err := this.suffixes.AppendToSQL(w, " ", args); err != nil {
			return err
		}
	}

	return nil
}

func (this *SelectBuilder) Count(args ...string) *SelectBuilder {
	var ts []string
	if this.foundRows {
		ts = []string{k_FOUND_ROWS}
	} else {
		ts = []string{k_COUNT}
	}

	if len(args) > 0 {
		ts = append(ts, args...)
	}
	var sb = this.Clone()
	sb.columns = statements{NewStatement(strings.Join(ts, " "))}
	sb.limit = nil
	sb.offset = nil
	return sb
}

// --------------------------------------------------------------------------------
func NewSelectBuilder() *SelectBuilder {
	var sb = &SelectBuilder{}
	sb.builder = newBuilder()
	sb.query = &query{b: sb}
	sb.scan = &scan{b: sb}
	return sb
}

// --------------------------------------------------------------------------------
func Select(columns ...string) *SelectBuilder {
	var sb = NewSelectBuilder()
	sb.Selects(columns...)
	return sb
}
