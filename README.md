[![MIT License](https://img.shields.io/github/license/glevap/sqlist)](LICENSE)

### Built upon the excellent [squirrel](https://github.com/Masterminds/squirrel) SQL builder.

<h1 style="border-bottom: 1px solid #ccc;">Sqlist</h1>
Build dynamic SQL queries for list views with filters, sorting, and cursor-based pagination for Go.

## Features

- 🔍 **Dynamic filters** — build WHERE clauses from HTTP query parameters
- 📊 **Sorting** — multiple sort fields with direction
- 📄 **Pagination** — offset/limit and cursor-based
- 🔗 **Joins** — support for all JOIN types
- 🎯 **Field mapping** — map request fields to database columns
- 🏗️ **Based on squirrel** — reliable and tested foundation

## Installation

```bash
go get github.com/glevap/sqlist
```

## Quick Start

```go
package main

import (
    "github.com/glevap/sqlist"
)

func main() {
    builder := sqlist.NewSQLBuilder().
        WithFrom("users u").
        WithFields("u.id", "u.name", "u.email").
        WithFieldConfig("name", "u.name", sqlist.ILike)

    // Apply filters from request
    builder.ApplyFilter("name", "john")
    
    // Build query
    result := builder.BuildSelect()
    sql, args := result.SQL, result.Args
    
    // Use with database
    // db.Query(sql, args...)
}
```


## License

MIT License - see [LICENSE](https://opensource.org/license/MIT).