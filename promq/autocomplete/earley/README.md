# `PromQL` Code Completion

`promq` offers code-completion in the interactive console. In order to do this,
we've chosen re-implement the `promql` parser.

 This wasn't a trivial decision but we did make this decision quite deliberately
 since the original `promql` parser is written using yacc, which is an LALR
 parser. LALR parsers are bottom-up parsers, which means that the syntax tree
 isn't actually assembled until the very end of the parse tree, which is a
 rather undesirable property to have when you want to offer suggestions on what
 may be valid syntax in the **middle** of a parsed string.

## Significant Considerations

There were a few significant aspects that factored into our choice of parser:

- Our choice should enable us to maintain parity with `promql` with relative
 ease, since we are writing a separate parser.
- Grammar rules should be definable top-down, since this makes it possible to
 anticipate possible valid syntactical structures during the parse.
- Since `promq` is a command-line tool, we expressly do not need to optimize for
  full-blown IDE level code-completion; our **implicit expectation** is that we
  will be interacting with smaller statements for the most part and that we
  would be re-parsing our tree incrementally.

## Earley Chart Parser

Given the factors at play, we converged around using an [earley chart parser
](https://en.wikipedia.org/wiki/Earley_parser). The more obvious downsides to
 this decision were namely:

 - The run-time can be really poor (n^3), if the [grammar is ambiguous](https://en.wikipedia.org/wiki/Ambiguous_grammar).
 - Choosing this (or any other parser beside the one provided in prometheus
 /prometheus), means that we are bifurcating `promql` for this tool.

 However, there are some counter considerations at play:

 - Despite the poor runtime:
    - even in the worst case we are dealing with input fed in by a command
     line, which means there is some practical upper bound for text length.
    - In the case of our parser, `n` is not directly determined by string
     length, since we feed our earley parser a stream of lexed tokens.
 - I'm not actually sure it is possible to avoid bifurcating `promql` for
   this kind of use-case, since `promql` uses an LALR parser. LR parsers, in
   general, support a restricted set of production rules which are not the
   most ideal for code completion (specifically, typically these sorts of
   rules will match many more things than are actually valid), since LR
   parsers don't actually construct the syntax tree until the very end of
   the parse.

[Earley chart parsers](https://en.wikipedia.org/wiki/Earley_parser) have a
 couple nice properties which makes it a particularly nice candidate for this
  usecase:

1. Earley parsers support all context free languages without restriction
, which means that it can support both LR and LL production rules.
2. Earley parsers naturally build and preserve state incrementally, which makes
   it trivial to convert into a more fully incremental parser (this is a desired
   optimization/property for our parser; i.e. batch-parsing isn't really on the
   table as an optimization).
3. Earley parsers are easy to implement and can also enable a setup such that it
   is easy to maintain parity with `promql` in the main [prometheus/prometheus codebase](https://github.com/prometheus/prometheus).

### Earley Internals

#### Earley algorithm

todo(YoyinZyc)

#### Incremental parsing

todo(logicalhan)

## Considered Alternatives

We considered a number of options for a parser:

### Hand-rolled Parser Heuristic
Our initial prototype hand-rolled the parser completely, using a novel (albeit
somewhat hacky) heuristic for determining grammar rules. The high level idea
was that we would lex the token stream and then move backward from the cursor
position to infer grammatical context and use that as a basis for suggesting
auto-completions.

For instance, given a string:

```
sum(metric{label=
```
We would read the "=" sign first then the token "label", followed by the "{"
and then "metric". Our heuristic would allow us to match grammar rules by the
first "=" sign, then filter this list by rules which permissibly accept tokens
which match the form 'label', then permissibly accept tokens which accept a
left brace, etc.

This heuristic was nice in that it ran in linear time, but abstracted poorly to
higher level grammar rules (i.e. of the form 'A = A + A | A + B').

### Using an LL(*) parser generator (ANTLR)

There are a couple downsides to using ANTLR:

- It uses java, which would mean we would need two languages (i.e. golang and
  java) in order to generate our support for a third language (promql). I
  personally find this aesthetically unsettling.
- While it supports LL-style production rules don't handle ambiguity well,
  which generally means it needs to choose a single parse path (you can handle
  this however during traversal).
