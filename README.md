# Welcome to toy-script

> **_NOTE_** I am doing this just for fun and learning.
> Please, please, please DON'T use this on production

toy-script is a lisp-like scripting language.

The parser and interpreter are written in go. Following <https://www.craftinginterpreters.com/> as a loose guide.

## Features

### primitive values

```
"string" # string
123      # int
false    # boolean
```

### statements

```
(@import
  (alias1 "path1") # fetches and tries to resolve the module
  (alias2 "path2") # imports are currently done at runtime
)

(@export
  var1 # only explicitly exported values can be imported
  var2 # only locally defined variables can be exported
)
```

#### expressions

anything apart from imports and exports is regarded as an expression

```
(identifier "string" 123 true (sub_expression))
```

#### function call

any expression that is NOT a pre-defined built-in is treated as a function call

```
(add 1 2 3 4)
(concat "a" "b" "c")
```

#### built-in data structs

lists

```
(@list 1 "b" "c" 4) # returns [1 2 3 4] as a slice
```

getting and setting in lists by index

```
(@get list index)
(@set list index value)
(@len list)
```

hashes

```
(@hash
  (key1 value)
  (key2 value)
) # returns a map[key]value
```

getting and setting in a hash

```
(@get hash key)
(@set hash key value)
(@len hash)
```

a stream is a way of handling async workloads
it is very similar to a list but it is lazy and hot
(values are not expected to be present at creation
and can be pushed in at any time)

```
(@stream value1 value2)
```

add values to the stream

```
(@push stream value3)
```

get values from the stream
returns the first available value in the stream

```
(@pull stream) # @pull is an alias for @get
```

end a stream
some built-in functions will block for the stream to close before running

```
(@close stream)
```

wait for a stream to finish

```
(@await stream)
```

get a value from any of the streams, whichever is ready first

```
(@select
  (@when stream1 (do_stuff_with value)) # value is set to the value of the stream
  (@when stream2 value)
)
```

> see below for some example use-cases

#### variables

```
(@var
  (name value_expression)
  (name value_expression)
)
```

functions are always annonimous

```
(@func (params) (body))
(@var (my_func (@func (params) (body))))
```

function bodies can contain 1 or more expressions
the last expression is returned

```
(@func () (
  (do_a)
  (do_b) # returns result of doB
))
```

#### loops

there are no traditional loops, only functional operations

filter - returns only members that satisfy the func
for streams returns a new stream

```
(@filter @func [@hash | @list | @stream])
```

map - transforms each memeber through the given func

```
# for streams returns a new stream
(@map @func [@hash | @list | @stream])
```

reduce - produces a result by going through all members
for streams - only returns once the stream is closed

```
(@reduce @func [@hash | @list | @stream] initial_value)
```

every - returns true if every member respects the condition
for streams - only returns once the stream is closed

```
(@every @func [@hash | @list | @stream])
```

any - returns true if any member respects the condition
for streams - only returns once the stream is closed

```
(@any @func [@hash | @list | @stream])
```

has - returns true if value found in struct, or false
for streams - only returns once the stream is closed

```
(@has [@hash | @list | @stream] value)
```

#### condition

```
(= 1 2) # returns false
(> 1 2) # returns false
(< 1 2) # returns true
(>= 1 2) # return false
```

```
(@match (cond_expression) # resolve the give expression and finds the first match
  (@when expected_value produced_value) # either a primitive value match
  (@when (match_expr) (then_expr)) # or matching an expression that produces true given the resolved cond value
  (@when expected_value (then_expr)) # these productions can be mixed
  (@when (match_expr) produced_value))
)
```

execute all expressions in a sequence, return the last
this is useful when a single expression is expected
and all values already exist in the current scope

```
(@seq
  (expr1)
  (expr2)
  (expr1)
)
```

returns a function that will executes all expressions
in the given sequence, passing subsequent results
to subsequent funcs, returns the result of the last func

```
(@chain
  func1
  func2
  func3
)
```

execute all expressions concurrently, return a stream
the go interpreter spawns a go routine for each expression

```
(@async
  (expr1)
  (expr2)
  (expr3)
)
```

## examples

### pub-sub pattern

spawn a go routine that maps over a list of urls
for each url - make an http request and parse the body as json
return a stream of the resulting values
(async converts the map result to a stream)

```
(@var my_data_stream (@async
  (@map (@func (url) ((@chain http.get json.parse))) list_of urls)
))
```

spawn a go routine that pulls a value from a single given stream, then prints it
in reality this will also return a stream, but it will contain only nil values

```
(@async (@chain
  (@pull my_data_stream)
  (stdio.print)
))
```

check out the examples/ dir for more examples
