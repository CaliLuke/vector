const functions = new Set(["ArrowFunctionExpression", "FunctionExpression", "FunctionDeclaration"]);
const branches = new Set([
  "IfStatement",
  "ConditionalExpression",
  "LogicalExpression",
  "SwitchStatement",
  "TryStatement",
  "ForStatement",
  "ForInStatement",
  "ForOfStatement",
  "WhileStatement",
]);
const errorSetters = new Set(["setError", "setErrors", "setFailure", "setSubmitError", "setErrorMap"]);
const browserActions = new Set(["webhook_secret_csv_download", "theme_config_download"]);

function browserActionReporters(program) {
  const reporters = new Set();
  for (const declaration of program.body) {
    if (declaration.type !== "ImportDeclaration" || declaration.source.value !== "@/helpers/browser-diagnostics")
      continue;
    for (const specifier of declaration.specifiers) {
      if (specifier.type === "ImportSpecifier" && name(specifier.imported) === "reportBrowserActionFailure")
        reporters.add(specifier.local.name);
    }
  }
  return reporters;
}

function name(node) {
  if (node?.type === "Identifier") return node.name;
  if (node?.type === "Literal") return node.value;
}

function member(node) {
  if (node?.type === "MemberExpression") return name(node.property);
}

function property(node, key) {
  if (node?.type !== "ObjectExpression") return;
  return node.properties.find(
    (entry) => entry.type === "Property" && (!entry.computed || entry.key.type === "Literal") && name(entry.key) === key
  )?.value;
}

function* calls(node, keys, unconditional = false) {
  if (!node || node.type === "CatchClause" || functions.has(node.type) || (unconditional && branches.has(node.type)))
    return;
  if (node.type === "CallExpression") yield node;
  for (const key of keys[node.type] ?? []) {
    const children = node[key];
    for (const child of Array.isArray(children) ? children : [children]) {
      yield* calls(child, keys, unconditional);
    }
  }
}

function isUIFailure(call) {
  const { callee, arguments: args } = call;
  if (name(callee) === "setToast") {
    const type = property(args[0], "type");
    return member(type) === "ERROR" || name(type) === "error";
  }
  if (name(callee.object) === "toast" && member(callee) === "error") return true;
  if (!errorSetters.has(name(callee)) && member(callee) !== "setErrorMap") return false;
  if (callee.type === "MemberExpression" && args[0]?.type === "ObjectExpression") {
    return args[0].properties.some((entry) => entry.type === "Property" && hasValue(entry.value));
  }
  return hasValue(args[0]);
}

function hasValue(node) {
  return (
    node != null &&
    !(node.type === "Literal" && [null, false, ""].includes(node.value)) &&
    !(node.type === "Identifier" && node.name === "undefined")
  );
}

function projectAccessPost(call) {
  const { callee, arguments: args } = call;
  return (
    (name(callee) === "fetch" || (name(callee.object) === "window" && member(callee) === "fetch")) &&
    name(args[0]) === "/api/diagnostics/project-access/" &&
    args[1]?.type === "ObjectExpression" &&
    args[1].properties.every((entry) => entry.type !== "SpreadElement") &&
    name(property(args[1], "method")) === "POST" &&
    name(property(args[1], "credentials")) === "omit" &&
    name(property(args[1], "keepalive")) === true
  );
}

function signal(call, reporters) {
  const { callee, arguments: args } = call;
  if (projectAccessPost(call) || (reporters.has(name(callee)) && browserActions.has(name(args[0]))))
    return "production";
  if (
    name(callee) === "reportLocalQueryError" ||
    (name(callee.object) === "console" && ["warn", "error"].includes(member(callee)))
  )
    return "local";
}

function signalOnPath(feedback, body, keys, reporters) {
  let local = false;
  for (let node = feedback; node !== body && node.parent; node = node.parent) {
    const parent = node.parent;
    const candidates = parent.type === "BlockStatement" ? parent.body : [node];
    for (const statement of candidates) {
      for (const call of calls(statement, keys, true)) {
        const signalType = signal(call, reporters);
        if (signalType === "production") return signalType;
        if (signalType === "local") local = true;
      }
    }
  }
  return local ? "local" : undefined;
}

function isPromiseCatch(node) {
  return (
    node.type === "CallExpression" &&
    member(node.callee) === "catch" &&
    // Effect.catch is a constructor for error recovery, not a Promise catch.
    name(node.callee.object) !== "Effect"
  );
}

function propagatesFailure(body, error) {
  if (error?.type !== "Identifier") return false;
  const last = body.type === "BlockStatement" ? body.body.at(-1) : undefined;
  if (last?.type === "ThrowStatement") {
    const thrown = last.argument;
    return (
      name(thrown) === error.name ||
      (thrown.type === "NewExpression" && name(property(thrown.arguments[1], "cause")) === error.name)
    );
  }
  const result = last?.type === "ReturnStatement" ? last.argument : body;
  return (
    result?.type === "CallExpression" &&
    name(result.callee.object) === "Promise" &&
    member(result.callee) === "reject" &&
    name(result.arguments[0]) === error.name
  );
}

function visibleFailureRule(target, message) {
  return {
    meta: { schema: [] },
    create(context) {
      const keys = context.sourceCode.visitorKeys;
      const reporters = browserActionReporters(context.sourceCode.ast);
      function check(body, error) {
        if (propagatesFailure(body, error)) return;
        for (const call of calls(body, keys)) {
          if (isUIFailure(call) && signalOnPath(call, body, keys, reporters) === target) {
            context.report({ node: call, message });
            return;
          }
        }
      }
      return {
        CatchClause(node) {
          check(node.body, node.param);
        },
        CallExpression(node) {
          const handler = node.arguments[0];
          if (isPromiseCatch(node) && functions.has(handler?.type)) check(handler.body, handler.params[0]);
        },
      };
    },
  };
}

function isEmptyHandler(handler) {
  if (!functions.has(handler?.type)) return false;
  const body = handler.body;
  if (body.type !== "BlockStatement") return isVoidValue(body);
  if (body.body.length === 0) return true;
  const [statement] = body.body;
  return body.body.length === 1 && statement.type === "ReturnStatement" && isVoidValue(statement.argument);
}

function isVoidValue(node) {
  return (
    node == null ||
    (node.type === "Identifier" && node.name === "undefined") ||
    (node.type === "UnaryExpression" && node.operator === "void" && node.argument.type === "Literal")
  );
}

function isDetached(node) {
  // Optional chaining inserts a ChainExpression around the promise call.
  const parent = node.parent.type === "ChainExpression" ? node.parent.parent : node.parent;
  return parent?.type === "UnaryExpression" && parent.operator === "void";
}

function ownsFailureElsewhere(source) {
  // TanStack submission adapters own their form state. Query invalidation and
  // cancellation are routine cache maintenance unless configured to reject.
  if (source.type !== "CallExpression") return false;
  if (projectAccessPost(source)) return true;
  const method = member(source.callee);
  if (method === "handleSubmit") return true;
  if (method === "cancelQueries") return true;
  if (method === "invalidateQueries") {
    return source.arguments.every(
      (argument) =>
        argument.type === "ObjectExpression" &&
        argument.properties.every((entry) => entry.type !== "SpreadElement") &&
        !hasValue(property(argument, "throwOnError"))
    );
  }
  return false;
}

export default {
  meta: { name: "vector-observability" },
  rules: {
    "unreported-ui-failure": visibleFailureRule(
      undefined,
      "This handler shows a failure without a diagnostic on the same path. Check for an existing reporter or record the operation and a safe error class."
    ),
    "local-only-ui-failure": visibleFailureRule(
      "local",
      "This handler shows a failure with console or local-only diagnostics. Check whether production records the operation and a safe error class."
    ),
    "swallowed-detached-rejection": {
      meta: { schema: [] },
      create(context) {
        return {
          CallExpression(node) {
            if (
              isPromiseCatch(node) &&
              isDetached(node) &&
              isEmptyHandler(node.arguments[0]) &&
              !ownsFailureElsewhere(node.callee.object)
            ) {
              context.report({
                node,
                message:
                  "This detached promise discards its rejection. Verify that the operation reports failure before it reaches this handler.",
              });
            }
          },
        };
      },
    },
  },
};
