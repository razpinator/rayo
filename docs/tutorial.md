# Tutorial: Building a Web JSON API and ETL Pipeline

This tutorial walks through building a simple web JSON API and a small ETL (Extract, Transform, Load) pipeline using Functure.

## Part 1: Web JSON API

We'll create a REST API that serves user data.

### Step 1: Define the API Structure

Create `examples/web/api.pygb`:

```python
import "functure/stdlib/http"

def main() {
    app = http.NewApp()

    # In-memory user store
    users = {
        "1": {"id": "1", "name": "Alice", "email": "alice@example.com"},
        "2": {"id": "2", "name": "Bob", "email": "bob@example.com"}
    }

    # GET /users - list all users
    app.Get("/users", func(ctx) {
        ctx.JSON(200, users)
    })

    # GET /users/:id - get user by ID
    app.Get("/users/:id", func(ctx) {
        id = ctx.Param("id")
        if id in users {
            ctx.JSON(200, users[id])
        } else {
            ctx.JSON(404, {"error": "User not found"})
        }
    })

    # POST /users - create new user
    app.Post("/users", func(ctx) {
        var data = ctx.BodyJSON()
        if data == None {
            ctx.JSON(400, {"error": "Invalid JSON"})
            return
        }
        id = str(len(users) + 1)
        users[id] = data
        data["id"] = id
        ctx.JSON(201, data)
    })

    app.Listen(":8080")
}
```

### Step 2: Transpile and Run

```sh
functurec transpile examples/web/api.pygb -o api.go
go run api.go
```

Test the API:

```sh
curl http://localhost:8080/users
curl http://localhost:8080/users/1
curl -X POST -H "Content-Type: application/json" -d '{"name":"Charlie","email":"charlie@example.com"}' http://localhost:8080/users
```

## Part 2: ETL Pipeline

We'll build a simple ETL that processes CSV data.

### Step 1: Define the ETL Logic

Create `examples/data/etl.pygb`:

```python
import "functure/stdlib/io"
import "functure/stdlib/data"

def main() {
    # Extract: Read CSV
    csv_data = io.ReadCSV("data/input.csv")

    # Transform: Filter and map
    processed = data.Filter(csv_data, func(row) {
        return row["active"] == "true"
    })
    processed = data.Map(processed, func(row) {
        return {
            "id": row["id"],
            "name": row["name"].upper(),
            "score": int(row["score"]) * 2
        }
    })

    # Load: Write JSON
    io.WriteJSON("data/output.json", processed)
    print("ETL completed")
}
```

### Step 2: Run the ETL

```sh
functurec transpile examples/data/etl.pygb -o etl.go
go run etl.go
```

## Next Steps

- Explore more examples in `/examples/`
- Read the [language spec](/docs/spec.md) for advanced features
- Contribute by implementing missing features or fixing bugs
