/** Keep server-evaluated formulas current at visible minute boundaries. */
export function nextFormulaClockBoundary() {
  return 60_000 - (Date.now() % 60_000);
}

/** Discovery catalogs deliberately omit formula expressions and dependencies. */
export function hasCustomPropertyFilter(expression: unknown): boolean {
  if (!expression || typeof expression !== "object") return false;
  if (Array.isArray(expression)) return expression.some(hasCustomPropertyFilter);
  return Object.entries(expression).some(
    ([key, value]) =>
      key.startsWith("customproperty_") ||
      ((key === "and" || key === "or" || key === "not") && hasCustomPropertyFilter(value))
  );
}
