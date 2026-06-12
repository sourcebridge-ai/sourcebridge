# Groovy tree-sitter node kinds

Discovery notes for the `github.com/smacker/go-tree-sitter/groovy` grammar
(bundled at `@v0.0.0-20240827094217-dd81d9e9be82`). These are the node
kinds the grammar actually produces — empirically verified, not assumed.
The queries in `languages_groovy.go`'s `groovyConfig()` are written against
these shapes.

## Imports

Node kind: `groovy_import` (NOT `import_declaration`).

| Source | Parse |
|---|---|
| `import foo.Bar` | `(groovy_import import: (qualified_name (identifier) (identifier)))` |
| `import foo.*` | `(groovy_import import: (qualified_name (identifier)) (wildcard_import))` |
| `import static foo.Bar.X` | `(groovy_import (modifier) import: (qualified_name (identifier) (identifier) (identifier)))` |

Query: `(groovy_import import: (qualified_name) @path) @import`

## Classes and methods

Class bodies are wrapped in a `closure` (not a typed class-body node):

```
class Example { void main(String[] args) { println("hi") } }
→
(class_definition name: (identifier)
  body: (closure
    (function_definition type: (builtintype) function: (identifier)
      parameters: (parameter_list
        (parameter type: (array_type (identifier)) name: (identifier)))
      body: (closure
        (function_call function: (identifier) args: (argument_list ...))))))
```

Class queries:
- `(class_definition name: (identifier) @name) @class`
- Method (class-scoped function): walk `class_definition → body: closure → function_definition`.
  `(class_definition body: (closure (function_definition function: (identifier) @name) @method))`

Bare (top-level) functions parse as `function_definition` directly:
```
def greet(String name) { println "hi $name" }
→
(function_definition function: (identifier)
  parameters: (...) body: (closure (juxt_function_call ...)))
```

So `FunctionQuery` doubles as top-level-def capture *and* method capture
if the outer context isn't constrained — `MethodQuery` adds the class
wrapper to distinguish.

## Calls

Two kinds: `function_call` (parenthesized, `foo(arg)`) and
`juxt_function_call` (juxtaposition / command chain, `plugins { ... }`).
Gradle DSL leans entirely on `juxt_function_call`.

Both take the same shape:
`(function_call function: (identifier) @callee args: (argument_list ...))`

CallQuery union both:
```
[
  (function_call function: (identifier) @callee) @call
  (juxt_function_call function: (identifier) @callee) @call
]
```

## Comments and groovydoc

Groovydoc `/** … */` parses as a **distinct node** — `groovy_doc` —
NOT a generic `(comment)`:

```
/** doc */
class X {}
→
(source_file (groovy_doc (first_line)) (class_definition ...))
```

Plain line/block comments (`//` and `/* */`) still emit `(comment)`.

DocCommentQuery should union both:
`[(groovy_doc) @comment (comment) @comment]`

## Known limitations (won't fix in v1)

### 1. Spock string-literal method names emit `ERROR` nodes

Spock's core test idiom is `def "should foo"() { … }`. The grammar
cannot parse this cleanly:

```
class FooSpec extends Specification {
  def "should foo"() { expect: true }
}
→
(class_definition … body: (closure
  (ERROR (function_call function: (string (string_content)) args: (argument_list))
         (closure (label name: (identifier)) (boolean_literal)))))
```

Consequence: **Spock method names are not captured as function symbols.**
Spock test detection therefore relies entirely on filename conventions
(`*Spec.groovy`, `src/test/groovy/**/*.groovy`), not on matching method
names via `TestFuncPattern`.

### 2. Multi-annotation classes emit `ERROR` nodes

```
@Transactional
@CompileStatic
class X {}
→
(source_file
  (declaration (annotation (identifier)) (ERROR) type: (identifier) name: (identifier))
  (juxt_function_call …))
```

The class name still extracts (as `name: (identifier)` inside the
`declaration` node), but the grammar's structural picture is broken
for multi-annotation cases. Single-annotation classes parse cleanly.

Consequence: ClassQuery also matches
`(declaration name: (identifier) @name)` as a fallback pattern, with
awareness that some context is lost.

### 3. Field-level symbols aren't extracted

Class fields (`Closure onChange = { … }`, `String name = "thing"`)
parse as `(declaration)` nodes inside the class body's `closure`.
The indexer doesn't extract fields as separate symbols today for
*any* language — this is by design, not a Groovy-specific gap.

### 4. .gvy files are not Grails-role-tagged

`.gvy` is a valid Groovy file extension and parses as Groovy, but
`GrailsRoleFor` does NOT assign a role to `.gvy` files even when they
appear under a Grails convention directory (e.g.
`grails-app/services/FooService.gvy`). Only `.groovy` (code dirs) and
`.gsp` (views) receive Grails roles. This is intentional: the Grails
convention contract is defined in terms of `.groovy` source files, and
`.gvy` usage in convention directories is rare enough that tagging it
would risk false-positives.

## References

- Grammar upstream: https://github.com/murtaza64/tree-sitter-groovy
- Smacker binding: `github.com/smacker/go-tree-sitter/groovy`
- Grammar's own tests (for reference node shapes):
  `$(go env GOMODCACHE)/github.com/smacker/go-tree-sitter@…/groovy/binding_test.go`
