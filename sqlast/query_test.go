package sqlast

import (
	"testing"

	"github.com/andreyvit/diff"
)

func TestSQLSelect_Eval(t *testing.T) {
	cases := []struct {
		name string
		in   *SQLSelect
		out  string
	}{
		{
			name: "simple select",
			in: &SQLSelect{
				Projection: []SQLSelectItem{
					&UnnamedExpression{
						Node: NewSQLIdentifier(NewSQLIdent("test")),
					},
				},
				Relation: &Table{
					Name: NewSQLObjectName("test_table"),
				},
			},
			out: "SELECT test FROM test_table",
		},
		{
			name: "join",
			in: &SQLSelect{
				Projection: []SQLSelectItem{
					&UnnamedExpression{
						Node: NewSQLObjectName("test"),
					},
				},
				Relation: &Table{
					Name: NewSQLObjectName("test_table"),
				},
				Joins: []*Join{
					{
						Relation: &Table{
							Name: NewSQLObjectName("test_table2"),
						},
						Op:       Inner,
						Constant: &NaturalConstant{},
					},
				},
			},
			out: "SELECT test FROM test_table NATURAL JOIN test_table2",
		},
		{
			name: "where",
			in: &SQLSelect{
				Projection: []SQLSelectItem{
					&UnnamedExpression{
						Node: NewSQLIdentifier(NewSQLIdent("test")),
					},
				},
				Relation: &Table{
					Name: NewSQLObjectName("test_table"),
				},
				Selection: &SQLBinaryExpr{
					Left: &SQLCompoundIdentifier{
						Idents: []*SQLIdent{NewSQLIdent("test_table"), NewSQLIdent("column1")},
					},
					Op:    Eq,
					Right: NewSingleQuotedString("test"),
				},
			},
			out: "SELECT test FROM test_table WHERE test_table.column1 = 'test'",
		},
		{
			name: "count and join",
			in: &SQLSelect{
				Projection: []SQLSelectItem{
					&ExpressionWithAlias{
						Expr: &SQLFunction{
							Name: NewSQLObjectName("COUNT"),
							Args: []ASTNode{&SQLCompoundIdentifier{
								Idents: []*SQLIdent{NewSQLIdent("t1"), NewSQLIdent("id")},
							}},
						},
						Alias: NewSQLIdent("c"),
					},
				},
				Relation: &Table{
					Name:  NewSQLObjectName("test_table"),
					Alias: NewSQLIdent("t1"),
				},
				Joins: []*Join{
					{
						Relation: &Table{
							Name:  NewSQLObjectName("test_table2"),
							Alias: NewSQLIdent("t2"),
						},
						Op: LeftOuter,
						Constant: &OnJoinConstant{
							Node: &SQLBinaryExpr{
								Left: &SQLCompoundIdentifier{
									Idents: []*SQLIdent{NewSQLIdent("t1"), NewSQLIdent("id")},
								},
								Op: Eq,
								Right: &SQLCompoundIdentifier{
									Idents: []*SQLIdent{NewSQLIdent("t2"), NewSQLIdent("test_table_id")},
								},
							},
						},
					},
				},
			},
			out: "SELECT COUNT(t1.id) AS c FROM test_table AS t1 LEFT JOIN test_table2 AS t2 ON t1.id = t2.test_table_id",
		},
		{
			name: "group by",
			in: &SQLSelect{
				Projection: []SQLSelectItem{
					&UnnamedExpression{
						Node: &SQLFunction{
							Name: NewSQLObjectName("COUNT"),
							Args: []ASTNode{NewSQLIdentifier(NewSQLIdent("customer_id"))},
						},
					},
					&QualifiedWildcard{
						Prefix: NewSQLObjectName("country"),
					},
				},
				Relation: &Table{
					Name: NewSQLObjectName("customers"),
				},
				GroupBy: []ASTNode{NewSQLIdentifier(NewSQLIdent("country"))},
			},
			out: "SELECT COUNT(customer_id), country.* FROM customers GROUP BY country",
		},
		{
			name: "having",
			in: &SQLSelect{
				Projection: []SQLSelectItem{
					&UnnamedExpression{
						Node: &SQLFunction{
							Name: NewSQLObjectName("COUNT"),
							Args: []ASTNode{NewSQLIdentifier(NewSQLIdent("customer_id"))},
						},
					},
					&UnnamedExpression{
						Node: NewSQLIdentifier(NewSQLIdent("country")),
					},
				},
				Relation: &Table{
					Name: NewSQLObjectName("customers"),
				},
				GroupBy: []ASTNode{NewSQLIdentifier(NewSQLIdent("country"))},
				Having: &SQLBinaryExpr{
					Op: Gt,
					Left: &SQLFunction{
						Name: NewSQLObjectName("COUNT"),
						Args: []ASTNode{NewSQLIdentifier(NewSQLIdent("customer_id"))},
					},
					Right: NewLongValue(3),
				},
			},
			out: "SELECT COUNT(customer_id), country FROM customers GROUP BY country HAVING COUNT(customer_id) > 3",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			act := c.in.Eval()

			if act != c.out {
				t.Errorf("must be \n%s but \n%s \n diff: %s", c.out, act, diff.CharacterDiff(c.out, act))
			}
		})
	}

}

func TestSQLQuery_Eval(t *testing.T) {
	cases := []struct {
		name string
		in   *SQLQuery
		out  string
	}{
		{
			// from https://www.postgresql.jp/document/9.3/html/queries-with.html
			name: "with cte",
			in: &SQLQuery{
				CTEs: []*CTE{
					{
						Alias: NewSQLIdent("regional_sales"),
						Query: &SQLQuery{
							Body: &SQLSelect{
								Projection: []SQLSelectItem{
									&UnnamedExpression{Node: NewSQLIdentifier(NewSQLIdent("region"))},
									&ExpressionWithAlias{
										Alias: NewSQLIdent("total_sales"),
										Expr: &SQLFunction{
											Name: NewSQLObjectName("SUM"),
											Args: []ASTNode{NewSQLIdentifier(NewSQLIdent("amount"))},
										},
									},
								},
								Relation: &Table{
									Name: NewSQLObjectName("orders"),
								},
								GroupBy: []ASTNode{NewSQLIdentifier(NewSQLIdent("region"))},
							},
						},
					},
				},
				Body: &SQLSelect{
					Projection: []SQLSelectItem{
						&UnnamedExpression{Node: NewSQLIdentifier(NewSQLIdent("product"))},
						&ExpressionWithAlias{
							Alias: NewSQLIdent("product_units"),
							Expr: &SQLFunction{
								Name: NewSQLObjectName("SUM"),
								Args: []ASTNode{NewSQLIdentifier(NewSQLIdent("quantity"))},
							},
						},
					},
					Relation: &Table{
						Name: NewSQLObjectName("orders"),
					},
					Selection: &SQLInSubQuery{
						Expr: NewSQLIdentifier(NewSQLIdent("region")),
						SubQuery: &SQLQuery{
							Body: &SQLSelect{
								Projection: []SQLSelectItem{
									&UnnamedExpression{Node: NewSQLIdentifier(NewSQLIdent("region"))},
								},
								Relation: &Table{
									Name: NewSQLObjectName("top_regions"),
								},
							},
						},
					},
					GroupBy: []ASTNode{NewSQLIdentifier(NewSQLIdent("region")), NewSQLIdentifier(NewSQLIdent("product"))},
				},
			},
			out: "WITH regional_sales AS (" +
				"SELECT region, SUM(amount) AS total_sales " +
				"FROM orders GROUP BY region) " +
				"SELECT product, SUM(quantity) AS product_units " +
				"FROM orders " +
				"WHERE region IN (SELECT region FROM top_regions) " +
				"GROUP BY region, product",
		},
		{
			name: "order by and limit",
			in: &SQLQuery{
				Body: &SQLSelect{
					Projection: []SQLSelectItem{
						&UnnamedExpression{Node: NewSQLIdentifier(NewSQLIdent("product"))},
						&ExpressionWithAlias{
							Alias: NewSQLIdent("product_units"),
							Expr: &SQLFunction{
								Name: NewSQLObjectName("SUM"),
								Args: []ASTNode{NewSQLIdentifier(NewSQLIdent("quantity"))},
							},
						},
					},
					Relation: &Table{
						Name: NewSQLObjectName("orders"),
					},
					Selection: &SQLInSubQuery{
						Expr: NewSQLIdentifier(NewSQLIdent("region")),
						SubQuery: &SQLQuery{
							Body: &SQLSelect{
								Projection: []SQLSelectItem{
									&UnnamedExpression{Node: NewSQLIdentifier(NewSQLIdent("region"))},
								},
								Relation: &Table{
									Name: NewSQLObjectName("top_regions"),
								},
							},
						},
					},
				},
				OrderBy: []*SQLOrderByExpr{
					{Expr: NewSQLIdentifier(NewSQLIdent("product_units"))},
				},
				Limit: NewLongValue(100),
			},
			out: "SELECT product, SUM(quantity) AS product_units " +
				"FROM orders " +
				"WHERE region IN (SELECT region FROM top_regions) " +
				"ORDER BY product_units LIMIT 100",
		},
		{
			name: "exists",
			in: &SQLQuery{
				Body: &SQLSelect{
					Projection: []SQLSelectItem{
						&UnnamedExpression{
							Node: &SQLWildcard{},
						},
					},
					Relation: &Table{
						Name: NewSQLObjectName("user"),
					},
					Selection: &SQLExists{
						Negated: true,
						Query: &SQLQuery{
							Body: &SQLSelect{
								Projection: []SQLSelectItem{
									&UnnamedExpression{
										Node: &SQLWildcard{},
									},
								},
								Relation: &Table{
									Name: NewSQLObjectName("user_sub"),
								},
								Selection: &SQLBinaryExpr{
									Op: And,
									Left: &SQLBinaryExpr{
										Op: Eq,
										Left: &SQLCompoundIdentifier{
											Idents: []*SQLIdent{
												NewSQLIdent("user"),
												NewSQLIdent("id"),
											},
										},
										Right: &SQLCompoundIdentifier{
											Idents: []*SQLIdent{
												NewSQLIdent("user_sub"),
												NewSQLIdent("id"),
											},
										},
									},
									Right: &SQLBinaryExpr{
										Op: Eq,
										Left: &SQLCompoundIdentifier{
											Idents: []*SQLIdent{
												NewSQLIdent("user_sub"),
												NewSQLIdent("job"),
											},
										},
										Right: NewSingleQuotedString("job"),
									},
								},
							},
						},
					},
				},
			},
			out: "SELECT * FROM user WHERE NOT EXISTS (" +
				"SELECT * FROM user_sub WHERE user.id = user_sub.id AND user_sub.job = 'job'" +
				")",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			act := c.in.Eval()

			if act != c.out {
				t.Errorf("must be \n%s but \n%s \n diff: %s", c.out, act, diff.CharacterDiff(c.out, act))
			}
		})
	}

}
