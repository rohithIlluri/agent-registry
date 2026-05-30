---
name: test-writer
description: "Write comprehensive unit, integration, and e2e tests. Detects framework and generates idiomatic tests with high coverage."
version: "2.0.1"
license: MIT
keywords: [testing, unit-tests, jest, pytest, vitest, coverage, tdd]
category: testing
allowed-tools: [Read, Write, Bash]
user-invocable: true
---

# Test Writer

Generate idiomatic, comprehensive tests for the specified code.

## When to invoke

Invoke when the user asks to "write tests", "add tests", or "improve coverage."

## Process

1. **Detect framework** — check `package.json` (jest/vitest/mocha), `pyproject.toml`/`setup.cfg` (pytest), `go.mod` (go test), `Gemfile` (rspec), etc.
2. **Read the source** — understand the function signatures, edge cases, and existing tests.
3. **Write tests** following the framework's idioms and conventions.
4. **Run the tests** — confirm they pass (or note expected failures).

## Coverage targets

- Happy path (normal inputs → expected outputs)
- Edge cases (empty, nil/null, zero, max values)
- Error cases (invalid input, expected panics/exceptions)
- Integration points (mocked dependencies)

## Framework cheat-sheet

### Jest / Vitest
```typescript
describe('functionName', () => {
  it('should return X when given Y', () => {
    expect(functionName(Y)).toBe(X)
  })
  it('should throw when input is invalid', () => {
    expect(() => functionName(bad)).toThrow(/message/)
  })
})
```

### pytest
```python
def test_function_name_returns_x_given_y():
    assert function_name(Y) == X

def test_function_name_raises_on_invalid():
    with pytest.raises(ValueError, match="message"):
        function_name(bad)
```

### Go
```go
func TestFunctionName(t *testing.T) {
    t.Run("returns X given Y", func(t *testing.T) {
        got := FunctionName(Y)
        if got != X { t.Errorf("got %v, want %v", got, X) }
    })
}
```

## Constraints

- Never write tests that always pass regardless of implementation.
- Use table-driven tests for multiple input/output pairs.
- Mock external I/O (DB, HTTP, filesystem) — do not depend on live services.
