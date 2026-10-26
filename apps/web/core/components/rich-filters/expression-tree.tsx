import type { TFilterExpression, TFilterProperty } from "@plane/types";
import { getDisplayOperator, isConditionNode, isNotGroupNode } from "@plane/utils/rich-filters";
import { X } from "lucide-react";
import type { TFilterExpressionEditor } from "./shared";
import { FilterItem } from "./filter-item/root";

type Props<P extends TFilterProperty> = {
  expression: TFilterExpression<P> | null;
  editor: TFilterExpressionEditor<P>;
  disabled: boolean;
};

export const FilterExpressionTree = function FilterExpressionBranch<P extends TFilterProperty>(props: Props<P>) {
  const { expression, editor, disabled } = props;
  if (!expression) return null;
  if (isConditionNode(expression)) {
    return (
      <FilterItem
        condition={{
          ...expression,
          operator: getDisplayOperator(expression.operator, editor.expression ?? expression, expression.id),
        }}
        editor={editor}
        isDisabled={disabled}
      />
    );
  }
  // Contiguous NOT wrappers on a leaf belong to that leaf's inclusion/exclusion
  // operator. NOT around a group must stay a visible group boundary.
  let leaf: TFilterExpression<P> = expression;
  while (isNotGroupNode(leaf)) leaf = leaf.child;
  if (isConditionNode(leaf)) {
    const condition = {
      ...leaf,
      operator: getDisplayOperator(leaf.operator, editor.expression ?? expression, leaf.id),
    };
    return <FilterItem editor={editor} condition={condition} isDisabled={disabled} />;
  }
  const excluded = isNotGroupNode(expression);
  return (
    <fieldset
      className="min-w-0 rounded-md border border-subtle px-2 pb-2"
      aria-label={excluded ? "Exclude matching filters" : "Filter group"}
    >
      <legend className="text-xs flex items-center gap-2 px-1 text-secondary">
        {isNotGroupNode(expression) ? (
          "Exclude matching filters"
        ) : (
          <label className="flex items-center gap-1">
            Match
            <select
              aria-label="Match conditions"
              value={expression.logicalOperator}
              disabled={disabled}
              onChange={(event) => {
                const value = event.target.value;
                if (value === "and" || value === "or") editor.onUpdateGroupOperator(expression.id, value);
              }}
              className="rounded border border-subtle bg-surface-1 px-1 py-0.5 text-primary"
            >
              <option value="and">all</option>
              <option value="or">any</option>
            </select>
          </label>
        )}
        <button
          type="button"
          aria-label="Remove filter group"
          disabled={disabled}
          onClick={() => editor.onRemoveCondition(expression.id)}
          className="rounded p-1 hover:bg-layer-2 disabled:opacity-50"
        >
          <X className="size-3" />
        </button>
      </legend>
      <div className="flex flex-wrap items-center gap-2">
        {isNotGroupNode(expression) ? (
          <FilterExpressionTree {...props} expression={expression.child} />
        ) : (
          expression.children.map((child) => <FilterExpressionTree key={child.id} {...props} expression={child} />)
        )}
      </div>
    </fieldset>
  );
};
