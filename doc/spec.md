# Rayo Language Specification v1.0

## Table of Contents

1. [Overview](#overview)
2. [Lexical Grammar](#lexical-grammar)
3. [Syntax Grammar](#syntax-grammar)
4. [Semantics](#semantics)
5. [Type System](#type-system)
6. [Go Transpilation Mapping](#go-transpilation-mapping)
7. [Examples](#examples)

## Overview

Rayo is a statically-typed, null-safe programming language that combines Python's familiar syntax with Go's performance and safety guarantees. It transpiles directly to Go 1.22+ code.

### Design Principles

- **Readability First**: Clear, unambiguous syntax
- **Null Safety**: Explicit optional types and safe navigation
- **Go Compatibility**: Direct, idiomatic Go transpilation
- **Python Familiarity**: Uses Python's reserved keywords and similar semantics

### Key Differences from Python

1. **Static Typing**: All variables must have declared or inferred types
2. **Curly Braces**: `{}` define code blocks, indentation is non-semantic
3. **Null Safety**: Explicit `None` handling with optional types
4. **Compilation**: Transpiles to Go, not interpreted

## Lexical Grammar

### Tokens

```ebnf
(* Comments *)
COMMENT = "//" {ANY_CHAR - NEWLINE} | "/*" {ANY_CHAR} "*/"

(* Literals *)
INTEGER = DIGIT {DIGIT}
FLOAT = DIGIT {DIGIT} "." DIGIT {DIGIT} [("e" | "E") ["+" | "-"] DIGIT {DIGIT}]
STRING = "\"" {STRING_CHAR} "\"" | "'" {STRING_CHAR} "'"
BOOLEAN = "True" | "False"
NULL = "None"

(* Identifiers *)
IDENTIFIER = (LETTER | "_") {LETTER | DIGIT | "_"}
LETTER = "a"..."z" | "A"..."Z"
DIGIT = "0"..."9"

(* Operators *)
PLUS = "+"
MINUS = "-"
MULTIPLY = "*"
DIVIDE = "/"
FLOOR_DIVIDE = "//"
MODULO = "%"
POWER = "**"
ASSIGN = "="
PLUS_ASSIGN = "+="
MINUS_ASSIGN = "-="
MULTIPLY_ASSIGN = "*="
DIVIDE_ASSIGN = "/="
MODULO_ASSIGN = "%="

(* Comparison *)
EQUAL = "=="
NOT_EQUAL = "!="
LESS = "<"
LESS_EQUAL = "<="
GREATER = ">"
GREATER_EQUAL = ">="

(* Logical *)
AND = "and"
OR = "or"
NOT = "not"

(* Bitwise *)
BIT_AND = "&"
BIT_OR = "|"
BIT_XOR = "^"
BIT_NOT = "~"
LEFT_SHIFT = "<<"
RIGHT_SHIFT = ">>"

(* Delimiters *)
LEFT_PAREN = "("
RIGHT_PAREN = ")"
LEFT_BRACE = "{"
RIGHT_BRACE = "}"
LEFT_BRACKET = "["
RIGHT_BRACKET = "]"
COMMA = ","
COLON = ":"
SEMICOLON = ";"
DOT = "."
QUESTION = "?"
ARROW = "->"

(* Safe Navigation *)
SAFE_DOT = "?."
SAFE_BRACKET = "?["

(* Whitespace *)
WHITESPACE = " " | "\t" | "\r"
NEWLINE = "\n"
```

### Operator Precedence (Highest to Lowest)

1. `()` `[]` `.` `?.` `?[` (postfix)
2. `**` (right-associative)
3. `+` `-` `~` `not` (unary)
4. `*` `/` `//` `%`
5. `+` `-` (binary)
6. `<<` `>>`
7. `&`
8. `^`
9. `|`
10. `==` `!=` `<` `<=` `>` `>=` `in` `not in` `is` `is not`
11. `and`
12. `or`
13. `=` `+=` `-=` `*=` `/=` `%=` (right-associative)

## Syntax Grammar

### Program Structure

```ebnf
program = {statement}

statement = declaration
          | assignment_statement
          | expression_statement
          | if_statement
          | while_statement
          | for_statement
          | try_statement
          | return_statement
          | break_statement
          | continue_statement
          | pass_statement
          | block_statement

block_statement = "{" {statement} "}"
```

### Declarations

```ebnf
declaration = function_declaration
            | class_declaration
            | variable_declaration

function_declaration = "def" IDENTIFIER "(" [parameter_list] ")" ["->" type_annotation] block_statement

parameter_list = parameter {"," parameter}
parameter = IDENTIFIER ":" type_annotation ["=" expression]

class_declaration = "class" IDENTIFIER ["(" IDENTIFIER ")"] block_statement

variable_declaration = IDENTIFIER ":" type_annotation ["=" expression]
```

### Statements

```ebnf
assignment_statement = target assignment_operator expression
assignment_operator = "=" | "+=" | "-=" | "*=" | "/=" | "%="
target = IDENTIFIER | attribute_access | subscript_access

if_statement = "if" expression block_statement 
               {"elif" expression block_statement} 
               ["else" block_statement]

while_statement = "while" expression block_statement

for_statement = "for" IDENTIFIER "in" expression block_statement

try_statement = "try" block_statement 
                {except_clause} 
                ["finally" block_statement]

except_clause = "except" [IDENTIFIER ["as" IDENTIFIER]] block_statement

return_statement = "return" [expression]
break_statement = "break"
continue_statement = "continue"
pass_statement = "pass"
```

### Expressions

```ebnf
expression = conditional_expression

conditional_expression = logical_or_expression ["if" logical_or_expression "else" conditional_expression]

logical_or_expression = logical_and_expression {"or" logical_and_expression}
logical_and_expression = equality_expression {"and" equality_expression}

equality_expression = relational_expression {("==" | "!=") relational_expression}
relational_expression = additive_expression {("<" | "<=" | ">" | ">=") additive_expression}

additive_expression = multiplicative_expression {("+" | "-") multiplicative_expression}
multiplicative_expression = power_expression {("*" | "/" | "//" | "%") power_expression}
power_expression = unary_expression ["**" power_expression]

unary_expression = ("+" | "-" | "not" | "~") unary_expression | postfix_expression

postfix_expression = primary_expression {postfix_operator}
postfix_operator = "[" expression "]"
                 | "?[" expression "]"
                 | "." IDENTIFIER
                 | "?." IDENTIFIER
                 | "(" [argument_list] ")"

primary_expression = IDENTIFIER
                   | literal
                   | "(" expression ")"
                   | list_literal
                   | dict_literal

literal = INTEGER | FLOAT | STRING | BOOLEAN | NULL

list_literal = "[" [expression_list] "]"
dict_literal = "{" [dict_entry_list] "}"
dict_entry_list = dict_entry {"," dict_entry}
dict_entry = expression ":" expression

expression_list = expression {"," expression}
argument_list = argument {"," argument}
argument = [IDENTIFIER "="] expression
```

### Type Annotations

```ebnf
type_annotation = basic_type | optional_type | list_type | dict_type

basic_type = "int" | "float" | "str" | "bool" | IDENTIFIER

optional_type = type_annotation "?"

list_type = "list" "[" type_annotation "]"

dict_type = "dict" "[" type_annotation "," type_annotation "]"
```

## Semantics

### Variable Scoping

- **Function Scope**: Variables declared in functions are local to that function
- **Class Scope**: Class members are accessible within the class
- **Global Scope**: Module-level variables
- **Block Scope**: Variables in `{}` blocks follow lexical scoping rules

### Null Safety

#### None Value
- `None` is the only null value
- Cannot be assigned to non-optional types
- Must be explicitly handled in optional types

#### Optional Types
```rayo
x: int? = None          // Optional int
y: str? = "hello"       // Optional string with value
z: int = None           // ERROR: Cannot assign None to non-optional
```

#### Safe Navigation
```rayo
obj?. attr              // Returns None if obj is None
obj?["key"]             // Returns None if obj is None
```

### Dictionary and Attribute Access

#### Disambiguation Rules

1. **Dot Access (`.`)**: Always attribute access first, then dict key if no attribute exists
2. **Bracket Access (`[]`)**: Always dictionary/list indexing
3. **Safe Access (`?.`, `?[]`)**: Same rules but returns `None` if object is `None`

```rayo
class Person {
    name: str
}

p: Person = Person()
d: dict[str, str] = {"name": "John"}

p.name          // Attribute access
d["name"]       // Dict key access
d.name          // ERROR: dict has no 'name' attribute (unless dynamically added)
```

### Error Handling

```rayo
try {
    // Code that may raise exceptions
    risky_operation()
} except ValueError as e {
    // Handle ValueError
    print("Value error:", e)
} except Exception {
    // Handle any other exception
    print("Unknown error")
} finally {
    // Always executed
    cleanup()
}
```

### Truthiness

Values that evaluate to `False`:
- `None`
- `False`
- `0` (int)
- `0.0` (float)
- `""` (empty string)
- `[]` (empty list)
- `{}` (empty dict)

All other values evaluate to `True`.

### Mutability

- **Immutable**: `int`, `float`, `str`, `bool`, tuples
- **Mutable**: `list`, `dict`, class instances
- **Assignment**: Always creates new binding for immutable types

### Evaluation Order

1. **Left-to-right** for most operators
2. **Short-circuit** for `and`, `or`
3. **Right-associative** for `=`, `**`

## Type System

### Built-in Types

```rayo
// Basic types
int: 64-bit signed integer
float: 64-bit floating point  
str: UTF-8 string
bool: True or False

// Collection types
list[T]: Dynamic array of type T
dict[K, V]: Hash map with key type K and value type V

// Special types
None: Null type
T?: Optional type T (can be T or None)
```

### Type Inference

```rayo
x := 42                 // Inferred as int
y := "hello"            // Inferred as str
z := [1, 2, 3]          // Inferred as list[int]
```

### Generics

```rayo
def identity[T](x: T) -> T {
    return x
}

class Container[T] {
    value: T
    
    def __init__(self, value: T) {
        self.value = value
    }
}
```

## Go Transpilation Mapping

### Types

| Rayo | Go |
|----------|-----|
| `int` | `int64` |
| `float` | `float64` |
| `str` | `string` |
| `bool` | `bool` |
| `None` | `nil` |
| `T?` | `*T` |
| `list[T]` | `[]T` |
| `dict[K, V]` | `map[K]V` |

### Null Safety

```rayo
// Rayo
x: int? = None
if x != None {
    print(x)
}
```

```go
// Generated Go
var x *int64 = nil
if x != nil {
    fmt.Println(*x)
}
```

### Safe Navigation

```rayo
// Rayo
result := obj?.method()?.field
```

```go
// Generated Go
var result *ReturnType
if obj != nil {
    if temp := obj.method(); temp != nil {
        result = &temp.field
    }
}
```

### Dictionary Access

```rayo
// Rayo
d: dict[str, int] = {"a": 1}
value := d["a"]
```

```go
// Generated Go
d := map[string]int64{"a": 1}
value := d["a"]  // Zero value if key doesn't exist
```

### Error Handling

```rayo
// Rayo
try {
    risky()
} except ValueError as e {
    handle(e)
}
```

```go
// Generated Go (using custom error types)
if err := risky(); err != nil {
    if valueErr, ok := err.(*ValueError); ok {
        handle(valueErr)
    } else {
        panic(err)  // Re-raise unhandled errors
    }
}
```

### Classes

```rayo
// Rayo
class Point {
    x: int
    y: int
    
    def __init__(self, x: int, y: int) {
        self.x = x
        self.y = y
    }
    
    def distance(self) -> float {
        return (self.x * self.x + self.y * self.y) ** 0.5
    }
}
```

```go
// Generated Go
type Point struct {
    X int64
    Y int64
}

func NewPoint(x, y int64) *Point {
    return &Point{X: x, Y: y}
}

func (p *Point) Distance() float64 {
    return math.Sqrt(float64(p.X*p.X + p.Y*p.Y))
}
```

## Examples

### Basic Function

```rayo
def fibonacci(n: int) -> int {
    if n <= 1 {
        return n
    }
    return fibonacci(n - 1) + fibonacci(n - 2)
}

result: int = fibonacci(10)
print(result)
```

### Null Safety Example

```rayo
def safe_divide(a: int, b: int) -> float? {
    if b == 0 {
        return None
    }
    return float(a) / float(b)
}

result: float? = safe_divide(10, 0)
if result != None {
    print("Result:", result)
} else {
    print("Division by zero")
}
```

### Class with Error Handling

```rayo
class BankAccount {
    balance: float
    
    def __init__(self, initial_balance: float) {
        if initial_balance < 0 {
            raise ValueError("Initial balance cannot be negative")
        }
        self.balance = initial_balance
    }
    
    def withdraw(self, amount: float) -> bool {
        if amount > self.balance {
            return False
        }
        self.balance -= amount
        return True
    }
    
    def deposit(self, amount: float) {
        if amount <= 0 {
            raise ValueError("Deposit amount must be positive")
        }
        self.balance += amount
    }
}

def main() {
    try {
        account: BankAccount = BankAccount(100.0)
        account.deposit(50.0)
        
        if account.withdraw(30.0) {
            print("Withdrawal successful")
        } else {
            print("Insufficient funds")
        }
        
    } except ValueError as e {
        print("Error:", e)
    }
}
```

### Dictionary and List Operations

```rayo
def process_data() {
    // Dictionary operations
    scores: dict[str, int] = {
        "Alice": 95,
        "Bob": 87,
        "Charlie": 92
    }
    
    // Safe access
    alice_score: int? = scores.get("Alice")  // Optional method
    dave_score: int = scores.get("Dave", 0)  // With default
    
    // List operations
    students: list[str] = ["Alice", "Bob", "Charlie"]
    
    for student in students {
        score: int? = scores.get(student)
        if score != None {
            print(f"{student}: {score}")
        }
    }
    
    // List comprehension equivalent
    high_scorers: list[str] = []
    for name, score in scores.items() {
        if score >= 90 {
            high_scorers.append(name)
        }
    }
}
```

### Generic Function

```rayo
def find_max[T](items: list[T], compare: def(T, T) -> bool) -> T? {
    if len(items) == 0 {
        return None
    }
    
    max_item: T = items[0]
    for item in items[1:] {
        if compare(item, max_item) {
            max_item = item
        }
    }
    return max_item
}

def compare_ints(a: int, b: int) -> bool {
    return a > b
}

numbers: list[int] = [1, 5, 3, 9, 2]
max_num: int? = find_max(numbers, compare_ints)
```

## Grammar Ambiguity Resolution

### Precedence Rules

1. **Attribute vs Dictionary**: `obj.key` is always attribute access
2. **Function Calls**: `f()` binds tighter than most operators
3. **Parentheses**: Override all precedence rules
4. **Type Annotations**: `x: int = 5` vs `x = (y: int)` - colon after identifier is always type annotation

### Context-Sensitive Parsing

1. **Class vs Function**: Determined by `class` vs `def` keyword
2. **Statement vs Expression**: Context determines interpretation
3. **Generic Brackets**: `func[T]` vs `func[0]` resolved by type context

## Future Considerations

### Planned Features
- Pattern matching
- Async/await syntax
- Module system
- Decorators
- Property getters/setters

### Transpilation Optimizations
- Dead code elimination
- Constant folding
- Inline small functions
- Escape analysis for pointer optimization

---

This specification defines a complete programming language that maintains Python's readability while providing Go's safety and performance characteristics. The grammar is unambiguous and suitable for implementation with standard parser generators.