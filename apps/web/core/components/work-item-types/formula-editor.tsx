import { useRef, useState, useLayoutEffect } from "react";
import { Button } from "@plane/propel/button";
import type { FormulaAuthoringInput, FormulaAuthoringResult } from "@/services/work-item-properties.service";
import { Choice, Field, inputStyle } from "./controls";
import type { PreviewContent, PreviewProperty } from "./preview-content";
import { formulaExamples, formulaFunctions, formulaReferences } from "./formula-reference";
import { FormulaPreview } from "./formula-preview";

export type FormulaPersistence = {
  scope: string;
  accountId?: string;
  revision: number;
  preview: (input: FormulaAuthoringInput, signal?: AbortSignal) => Promise<FormulaAuthoringResult>;
};

export function FormulaEditor({
  property,
  library,
  onChange,
  persistence,
}: {
  property: PreviewProperty;
  library: Pick<PreviewContent, "types" | "properties" | "projects">;
  onChange: (expression: string, result: string) => void;
  persistence?: FormulaPersistence;
}) {
  const [typeId, setTypeId] = useState(
    library.types.find((type) => type.properties.includes(property.id))?.id ?? library.types[0]?.id ?? ""
  );
  const editor = useRef<HTMLTextAreaElement>(null);
  const insertedCursor = useRef<number | undefined>(undefined);
  useLayoutEffect(() => {
    if (insertedCursor.current === undefined) return;
    editor.current?.focus();
    editor.current?.setSelectionRange(insertedCursor.current, insertedCursor.current);
    insertedCursor.current = undefined;
  });
  const references = formulaReferences(property.id, typeId, library.types, library.properties);
  const insert = (reference: string) => {
    const start = editor.current?.selectionStart ?? property.expression.length;
    const end = editor.current?.selectionEnd ?? start;
    const token = `{{${reference}}}`;
    const expression = property.expression.slice(0, start) + token + property.expression.slice(end);
    const cursor = start + token.length;
    if (expression === property.expression) {
      insertedCursor.current = undefined;
      editor.current?.focus();
      editor.current?.setSelectionRange(cursor, cursor);
      return;
    }
    insertedCursor.current = cursor;
    onChange(expression, property.result);
  };
  return (
    <div className="space-y-4">
      <Field
        label="Reference type"
        hint="Custom references must belong to this type and every type already using this formula."
      >
        {(id) => (
          <select id={id} className={inputStyle} value={typeId} onChange={(event) => setTypeId(event.target.value)}>
            {library.types.map((type) => (
              <option key={type.id} value={type.id}>
                {type.name}
              </option>
            ))}
          </select>
        )}
      </Field>
      <Field label="Expression" hint="Insert fields by their stable reference. Formula values are read-only.">
        {(id) => (
          <textarea
            ref={editor}
            id={id}
            rows={4}
            spellCheck={false}
            className={`${inputStyle} font-mono`}
            value={property.expression}
            onChange={(event) => onChange(event.target.value, property.result)}
          />
        )}
      </Field>
      <Field label="Insert field">
        {(id) => (
          <select
            id={id}
            className={inputStyle}
            value=""
            onChange={(event) => {
              if (event.target.value) insert(event.target.value);
            }}
          >
            <option value="">Choose a field…</option>
            {references.map((field) => (
              <option key={field.name} value={field.name}>
                {field.title} · {field.kind} · {`{{${field.name}}}`}
              </option>
            ))}
          </select>
        )}
      </Field>
      <Field label="Result type">
        {() => (
          <Choice
            label="Result type"
            value={property.result}
            options={["Number", "Text", "Date", "Boolean"]}
            onChange={(result) => onChange(property.expression, result)}
          />
        )}
      </Field>
      <div className="flex flex-wrap gap-2" aria-label="Formula examples">
        {formulaExamples.map((example) => (
          <Button
            key={example.expression}
            variant="secondary"
            onClick={() => onChange(example.expression, example.result)}
          >
            {example.expression}
          </Button>
        ))}
      </div>
      <details className="rounded-md border border-subtle p-3 text-12">
        <summary className="cursor-pointer font-medium">Functions and operators</summary>
        <dl className="mt-3 space-y-3">
          {formulaFunctions.map((fn) => (
            <div key={fn.signature}>
              <dt className="font-mono">{fn.signature}</dt>
              <dd className="mt-1 text-secondary">{fn.help}</dd>
            </div>
          ))}
        </dl>
        <p className="mt-3 text-secondary">
          Use + − * / for numbers, &amp; to concatenate, and = != &lt; &lt;= &gt; &gt;= to compare. Dates can add or
          subtract whole days; subtracting dates returns days. Empty inputs produce an empty result. Formula references
          cannot point to another formula.
        </p>
      </details>
      {persistence && (
        <FormulaPreview
          key={typeId}
          property={property}
          typeId={typeId}
          projects={library.projects}
          persistence={persistence}
        />
      )}
    </div>
  );
}
