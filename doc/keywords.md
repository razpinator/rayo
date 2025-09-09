# Functure Keywords Reference

This document lists all reserved keywords in Functure, which are identical to Python's reserved keywords. These keywords cannot be used as identifiers (variable names, function names, class names, etc.).

## Complete List of Reserved Keywords

The following 35 keywords are reserved in Functure:

### Control Flow Keywords
- `if` - Conditional statement
- `elif` - Else-if conditional
- `else` - Else clause
- `while` - While loop
- `for` - For loop
- `break` - Exit loop
- `continue` - Skip to next iteration
- `pass` - No-operation statement

### Function and Class Keywords
- `def` - Function definition
- `class` - Class definition
- `return` - Return from function
- `yield` - Generator yield (reserved for future use)
- `lambda` - Anonymous function (reserved for future use)

### Exception Handling Keywords
- `try` - Begin exception handling block
- `except` - Exception handler
- `finally` - Always-executed cleanup block
- `raise` - Raise an exception
- `assert` - Assertion statement (reserved for future use)

### Logical Operators
- `and` - Logical AND
- `or` - Logical OR
- `not` - Logical NOT

### Value Keywords
- `True` - Boolean true value
- `False` - Boolean false value
- `None` - Null/None value

### Import Keywords (reserved for future module system)
- `import` - Import module
- `from` - Import from module
- `as` - Alias in import

### Scope Keywords
- `global` - Global variable declaration (reserved)
- `nonlocal` - Non-local variable declaration (reserved)

### Context Management (reserved for future use)
- `with` - Context manager

### Comparison Keywords
- `in` - Membership test
- `is` - Identity comparison

### Deletion Keyword (reserved)
- `del` - Delete statement (reserved for future use)

### Async Keywords (reserved for future async support)
- `async` - Asynchronous function definition (reserved)
- `await` - Await asynchronous operation (reserved)

## Keyword Usage in Functure

### Currently Implemented

The following keywords are actively used in Functure v1.0:

```functure
// Control flow
if condition { ... }
elif other_condition { ... }
else { ... }

while condition { ... }

for item in collection { ... }

break
continue
pass

// Functions and classes  
def function_name() -> return_type { ... }
class ClassName { ... }
return value

// Exception handling
try { ... }
except ExceptionType as e { ... }
finally { ... }
raise Exception("message")

// Logical operators
result := a and b
result := a or b  
result := not condition

// Values
flag := True
flag := False
value := None

// Comparison
if item in collection { ... }
if obj1 is obj2 { ... }
```

### Reserved for Future Use

These keywords are reserved but not yet implemented:

```functure
// Module system (planned)
import module_name
from module_name import item
import module_name as alias

// Generators (planned)
def generator() {
    yield value
}

// Lambda functions (planned)
callback := lambda x: x * 2

// Variable scope (planned)
global variable_name
nonlocal variable_name  

// Context managers (planned)
with resource as r {
    // use resource
}

// Deletion (planned)
del variable_name

// Async/await (planned)
async def async_function() {
    result := await other_async_function()
}

// Assertions (planned)
assert condition, "error message"
```

## Soft Keywords

Unlike Python, Functure does not currently have soft keywords. All keywords in the above list are hard keywords and cannot be used as identifiers in any context.

## Case Sensitivity

All keywords in Functure are case-sensitive and must be written exactly as shown:

```functure
// Correct
if True { ... }
def my_function() { ... }

// Incorrect - these are not keywords
If True { ... }        // 'If' is not a keyword
DEF my_function() { ... }  // 'DEF' is not a keyword
```

## Keyword vs. Built-in Functions

Note the distinction between keywords and built-in functions:

### Keywords (Reserved)
- `True`, `False`, `None` - These are keywords, not identifiers
- Cannot be redefined or shadowed

### Built-in Functions (Not Reserved)
- `print()`, `len()`, `str()`, `int()`, `float()` - These are built-in functions
- Can be shadowed (though not recommended)

```functure
// This is allowed but not recommended
def print(msg: str) {
    // Custom print function shadows built-in
}
```

## Implementation Notes

### Parser Considerations

When implementing a parser for Functure:

1. **Keyword Recognition**: Keywords should be recognized before identifier tokenization
2. **Context Independence**: Keywords are recognized regardless of context
3. **Unicode**: Keywords are ASCII-only and do not support Unicode variations
4. **Lookahead**: Some keywords may require lookahead to disambiguate (e.g., `def` vs identifier starting with "def")

### Lexer Implementation

```go
// Example Go code for keyword recognition
var keywords = map[string]TokenType{
    "and":      TOKEN_AND,
    "as":       TOKEN_AS,
    "assert":   TOKEN_ASSERT,
    "async":    TOKEN_ASYNC,
    "await":    TOKEN_AWAIT,
    "break":    TOKEN_BREAK,
    "class":    TOKEN_CLASS,
    "continue": TOKEN_CONTINUE,
    "def":      TOKEN_DEF,
    "del":      TOKEN_DEL,
    "elif":     TOKEN_ELIF,
    "else":     TOKEN_ELSE,
    "except":   TOKEN_EXCEPT,
    "False":    TOKEN_FALSE,
    "finally":  TOKEN_FINALLY,
    "for":      TOKEN_FOR,
    "from":     TOKEN_FROM,
    "global":   TOKEN_GLOBAL,
    "if":       TOKEN_IF,
    "import":   TOKEN_IMPORT,
    "in":       TOKEN_IN,
    "is":       TOKEN_IS,
    "lambda":   TOKEN_LAMBDA,
    "None":     TOKEN_NONE,
    "nonlocal": TOKEN_NONLOCAL,
    "not":      TOKEN_NOT,
    "or":       TOKEN_OR,
    "pass":     TOKEN_PASS,
    "raise":    TOKEN_RAISE,
    "return":   TOKEN_RETURN,
    "True":     TOKEN_TRUE,
    "try":      TOKEN_TRY,
    "while":    TOKEN_WHILE,
    "with":     TOKEN_WITH,
    "yield":    TOKEN_YIELD,
}
```

## Migration from Python

When porting Python code to Functure, be aware that:

1. All Python keywords remain keywords in Functure
2. Some keywords may have slightly different semantics (documented in main spec)
3. New syntax features use curly braces instead of indentation
4. Type annotations are required where not inferred

This maintains maximum compatibility with Python's keyword system while enabling the additional features that Functure provides.