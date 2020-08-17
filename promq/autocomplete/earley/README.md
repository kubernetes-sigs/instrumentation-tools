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

There are some key terms in earley algorithm.
1. Terminal and non-terminal symbols: In promql.go, we defined a lot of terminal and non-terminal symbols. 
They are the lexical elements used in specifying the grammar rules(production rules). 
Terminal symbols are the elementary symbols of the grammar. 
Non-terminal symbols can be replaced by groups of terminal symbols according to the grammar rules.
2. Grammar rule: In promql.go, we also defined the grammar rules. The left of the rules can be replaced by the right. 
There is one root rule which is the start point for any input string.
3. Input position: The parser will the parse the input string from left to right and step by step. 
It will start from the position prior to the input string which is the position 0. 
The last position is the position after the last token.
4. State set: For every input position, the parser generates a state set. The state set at input position k is called S(k).
A state set is composed by a list of state. `EarleyItem` in item.go represent a single state.
Each state is a tuple (X -> a  •  b, i):
    - The dot represents the input position. Here the dot is at position 1.
    - The state applies rule X -> ab
    - i is the origin position where the the matching of the production began.
5. Chart: Chart records the state set of each position. The length of earley chart is `len(input_string) + 1`

Earley parser starts from position 0 with root grammar rule and then repeatedly executes three operations: prediction, scanning and completion:
- Prediction: For every state in S(k) of the form (X → α • Y β, j) (where j is the origin position as above), add (Y → • γ, k) to S(k) for every production in the grammar with Y on the left-hand side (Y → γ).
- Scanning: If a is the next symbol in the input stream, for every state in S(k) of the form (X → α • a β, j), add (X → α a • β, j) to S(k+1).
- Completion: For every state in S(k) of the form (Y → γ •, j), find all states in S(j) of the form (X → α • Y β, i) and add (X → α Y • β, i) to S(k).

Only new items will be added to state set. Two states with same rule, dot position and origin position will be viewed as duplicate states.

We use the last state(the dot is at the end of input string) to generate completion suggestion. In each unfinished state(there are tokens after dot) of the state set, 
if the symbol after dot is a terminal, then it is a potential suggestion. The suggestion generated from earley algorithm is a suggested token type. 
There is a mapping in `completer.go` which matches type to real suggested string.

#### Incremental parsing

Earley parser can have poor runtime sometimes. To optimize, we will not parse input string every time from beginning. 
For example, if the previous input is 
`sum(metric_name_one{` and the current input is `sum(metric_name_one)`, 
parser will keep the previous states that cover `sum(metric_name_one` and start parsing from the next one. Even if the previous input and new input are different from the first token, 
the first state set can be reused because the first input position is always prior to the input string. 

This incremental parsing is really practical because user keeps inputting word by word and the later state set can always build on top of previous state sets.

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
