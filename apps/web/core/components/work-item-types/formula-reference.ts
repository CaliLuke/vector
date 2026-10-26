import type { PreviewProperty, PreviewType } from "./preview-content";

const builtins = [
  { name: "name", title: "Work item name", kind: "Text" },
  { name: "priority", title: "Priority", kind: "Text" },
  { name: "estimate_point", title: "Estimate", kind: "Number" },
  { name: "start_date", title: "Start date", kind: "Date" },
  { name: "due_date", title: "Due date", kind: "Date" },
];

export function formulaReferences(
  propertyId: string,
  typeId: string,
  types: PreviewType[],
  properties: PreviewProperty[]
) {
  const selected = types.find((type) => type.id === typeId);
  if (!selected) return builtins;
  const scopes = [selected, ...types.filter((type) => type.id !== typeId && type.properties.includes(propertyId))];
  return [
    ...builtins,
    ...properties
      .filter(
        (property) =>
          property.id !== propertyId &&
          property.internalName &&
          ["Text", "Number", "Boolean", "Date"].includes(property.kind) &&
          scopes.every((type) => type.properties.includes(property.id))
      )
      .map((property) => ({ name: property.internalName!, title: property.name, kind: property.kind })),
  ];
}

export const formulaExamples = [
  { expression: "LEN({{name}})", result: "Number" },
  { expression: 'IF({{priority}} = "URGENT", "Urgent", "Normal")', result: "Text" },
  { expression: "TODAY()", result: "Date" },
];

export const formulaFunctions = [
  {
    signature: "IF(condition, then, else)",
    help: "A Boolean condition chooses one branch. Both branches must return the same kind; only the chosen branch is evaluated.",
  },
  {
    signature: "ROUND(number, digits = 0)",
    help: "Round half away from zero. Digits must be an integer from −15 to 15.",
  },
  { signature: "ABS(number)", help: "Return the absolute numeric value." },
  { signature: "UPPER(text), LOWER(text)", help: "Change text case." },
  { signature: "LEN(text)", help: "Count Unicode characters." },
  {
    signature: "CONCAT(value, …)",
    help: "Combine one or more scalar values as text. Numbers use locale-independent formatting; dates use ISO format.",
  },
  { signature: "TODAY(), NOW()", help: "Return the UTC date or timestamp at the displayed evaluation time." },
];
