import type { TWorkItemFilterConditionData, TWorkItemFilterExpression } from "@plane/types";

// Children precede their parent, so the draft has no recursive field paths.
// The form owns this editable representation; API payloads remain nested expressions.
// Remove the projection when native recursive fields compile: TanStack/form#1830.
export type FilterDraftNode =
  | { kind: "condition"; value: TWorkItemFilterConditionData }
  | { kind: "and" | "or"; children: number[] }
  | { kind: "not"; child: number }
  | { kind: "merge"; group: number; siblings: number };

export function createFilterDraft(expression: TWorkItemFilterExpression): FilterDraftNode[] {
  const nodes: FilterDraftNode[] = [];
  const withSiblings = (node: FilterDraftNode, siblings: TWorkItemFilterExpression): number => {
    const group = nodes.push(node) - 1;
    if (Object.keys(siblings).length === 0) return group;
    return nodes.push({ kind: "merge", group, siblings: visit(siblings) }) - 1;
  };
  const visit = (value: TWorkItemFilterExpression): number => {
    if ("and" in value && value.and) {
      const { and, ...siblings } = value;
      return withSiblings({ kind: "and", children: and.map(visit) }, siblings);
    }
    if ("or" in value && value.or) {
      const { or, ...siblings } = value;
      return withSiblings({ kind: "or", children: or.map(visit) }, siblings);
    }
    if ("not" in value && value.not) {
      const { not, ...siblings } = value;
      return withSiblings({ kind: "not", child: visit(not) }, siblings);
    }
    return nodes.push({ kind: "condition", value }) - 1;
  };
  visit(expression);
  return nodes;
}

export function filterDraftExpression(nodes: FilterDraftNode[]): TWorkItemFilterExpression {
  const expressions: TWorkItemFilterExpression[] = [];
  const childExpression = (index: number): TWorkItemFilterExpression => {
    const expression = expressions[index];
    if (!expression) throw new Error("Filter draft references a missing child");
    return expression;
  };
  for (const node of nodes) {
    switch (node.kind) {
      case "condition":
        expressions.push(node.value);
        break;
      case "not":
        expressions.push({ not: childExpression(node.child) });
        break;
      case "and":
        expressions.push({ and: node.children.map(childExpression) });
        break;
      case "or":
        expressions.push({ or: node.children.map(childExpression) });
        break;
      case "merge":
        expressions.push({ ...childExpression(node.group), ...childExpression(node.siblings) });
        break;
    }
  }
  return expressions.at(-1) ?? {};
}
