# M1-003: Limit Sticky HTML Element Depth

Status: Review ready for the M1 Go cutover.

## Source behavior

Django uses nh3 0.2.18 for Sticky HTML. The source accepts 4,096 and 4,097 nested `div` elements.

The source also accepts at least 1,024 nested allowed elements. The source has a 10 MiB input limit but no element-depth limit.

## Go behavior

The Go validator accepts a maximum element depth of 4,096. It returns `ErrHTMLTooDeep` for the next element.

The parser stops before it grows the open-element stack beyond 4,096. The error does not include request data.

An iterative tree check runs before the recursive sanitizer and renderer. Thus, these functions do not receive a tree above the limit.

## Reason for the difference

A 10 MiB request can contain approximately one million nested short elements. An unbounded tree can exhaust memory or the Go stack.

The 4,096 limit is four times the largest source depth in the main compatibility fixtures. A 4,096-element `div` fixture is 45,060 bytes.

This limit keeps a large margin for valid editor content. It also gives a fixed memory and stack boundary for nested elements.

## Regression evidence

`html-deep-valid-nesting.json` proves that nh3 accepts 4,096 and 4,097 nested `div` elements.

The Go fixture test accepts the 4,096-element input. It rejects the 4,097-element input with `ErrHTMLTooDeep` and no output.

The parser test proves that the guard stops the open-element stack. The tree test proves the pre-sanitizer depth check.

Existing fixtures continue to prove HTML5 repair and unsafe-content removal below the limit.

## Frontend effect

Ordinary Sticky content does not change. A request above the limit will receive a safe validation error after the Go route cutover.

The Django route remains active while `GO_STICKIES_ENABLED=false`. This difference starts when the Go Sticky route is active.
