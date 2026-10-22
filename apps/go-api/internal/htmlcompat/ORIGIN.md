# HTML5 Parser Origin

This package contains the HTML parser from `golang.org/x/net/html` v0.57.0.

The source uses the BSD license in `LICENSE`. The copied parser tests use the upstream html5lib data.

Plane replaces the 512-open-element guard with the `ParseOptionMaxElementDepth` option. The Sticky sanitizer sets the option to 4,096 elements.

The parser preserves the typed `ErrElementDepthLimit` error. It stops before it grows the open-element stack beyond the configured limit.

Keep all other parser behavior and test data identical to v0.57.0. Review the complete upstream diff before each version change.
