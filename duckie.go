package main

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"slices"

	"github.com/xwb1989/sqlparser"

	_ "github.com/marcboeker/go-duckdb"
)

type Record []interface{}

type Page struct {
	SelectedSource string
	DataSources    []string
	Query          string
	Results        []Record
	Columns        []string
}

var dataSources = []string{"call_center.csv", "catalog_page.csv"}

var describeColumns = []string{"column_name", "column_type", "null", "key", "default", "extra"}

func runQuery(query string, sel_count int) ([]Record, error) {
	db, err := sql.Open("duckdb", "?threads=4")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	results := []Record{}
	for rows.Next() {
		values := make([]interface{}, sel_count)
		for i := range values {
			values[i] = new(interface{})
		}
		rows.Scan(values...)
		results = append(results, values)
	}
	return results, nil
}

func sourceExists(selection string) bool {
	return slices.Contains(dataSources, selection)
}

func describeSource(dataSource string) ([]Record, error) {
	query := fmt.Sprintf("DESCRIBE SELECT * FROM %s", dataSource)
	results, err := runQuery(query, len(describeColumns))
	if err != nil {
		return nil, err
	}
	return results, nil
}

func initPage(selectedSource string) (*Page, error) {
	if len(selectedSource) > 0 && !sourceExists(selectedSource) {
		return nil, errors.New("selected data source does not exist")
	}
	return &Page{SelectedSource: selectedSource, DataSources: dataSources}, nil
}

func queryPageHandler(w http.ResponseWriter, r *http.Request, title string) {
	dataSource := r.URL.Query().Get("datasource")
	page, err := initPage(dataSource)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if len(dataSource) > 0 {
		// Data source is selected, so let's run DESCRIBE
		page.Query = fmt.Sprintf("DESCRIBE SELECT * FROM %s", dataSource)
		results, err := describeSource(dataSource)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		page.Results = results
	} else {
		page.Query = "Please choose data source from above."
	}
	page.Columns = describeColumns
	renderTemplate(w, "query", page)
}

func queryHandler(w http.ResponseWriter, r *http.Request, title string) {
	selectedSource := r.FormValue("datasource")
	page, err := initPage(selectedSource)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// Parse query
	q := r.FormValue("query")
	stmt, err := sqlparser.Parse(q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check SELECT statement
	select_stmt, ok := stmt.(*sqlparser.Select)
	if !ok {
		http.Error(w, "Not a SELECT statement.", http.StatusInternalServerError)
		return
	}

	// Get columns so we can allocate memory for results
	columns := []string{}
	for _, expr := range select_stmt.SelectExprs {
		switch col := expr.(type) {
		case *sqlparser.AliasedExpr:
			// Try to assert ColName type
			if colName, ok := col.Expr.(*sqlparser.ColName); ok {
				columns = append(columns, colName.Name.String())
			} else {
				// Expressions like COUNT(*), CONCAT(name, last_name), etc.
				columns = append(columns, sqlparser.String(col.Expr))
			}
		case *sqlparser.StarExpr:
			// Case of `SELECT *` so let's get all columns from DESCRIBE
			describeResults, err := describeSource(selectedSource)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			for _, describeEntry := range describeResults {
				v := reflect.ValueOf(describeEntry[0]).Elem()
				columns = append(columns, v.Interface().(string))
			}
		}
	}

	results, err := runQuery(q, len(columns))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	page.Query = q
	page.Columns = columns
	page.Results = results
	page.SelectedSource = selectedSource
	renderTemplate(w, "query", page)
}

var templates = template.Must(template.ParseFiles("query.html"))

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var validPath = regexp.MustCompile("^(/|/query)$")

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[0])
	}
}

func main() {
	http.HandleFunc("/", makeHandler(queryPageHandler))
	http.HandleFunc("/query", makeHandler(queryHandler))
	log.Fatal(http.ListenAndServe(":8081", nil))
}
