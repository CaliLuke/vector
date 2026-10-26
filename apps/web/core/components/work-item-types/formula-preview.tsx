import { FormulaSamplePicker } from "./formula-sample-picker";
import { useState } from "react";
import { skipToken, useQuery } from "@tanstack/react-query";
import { Button } from "@plane/propel/button";
import type { FormulaAuthoringResult } from "@/services/work-item-properties.service";
import { typeErrorMessage } from "@/services/work-item-types.service";
import { propertyToSave } from "./property-adapter";
import type { PreviewProperty } from "./preview-content";
import type { FormulaPersistence } from "./formula-editor";
import { Field, inputStyle } from "./controls";

export function FormulaPreview({
  property,
  typeId,
  projects,
  persistence,
}: {
  property: PreviewProperty;
  typeId: string;
  projects: { id: string; name: string }[];
  persistence: FormulaPersistence;
}) {
  const [projectId, setProjectId] = useState("");
  const [sample, setSample] = useState<{ id: string; name: string }>();
  const [requested, setRequested] = useState("");
  const command = {
    revision: persistence.revision,
    definition: propertyToSave(property),
    type_id: typeId,
    project_id: projectId || undefined,
    issue_id: sample?.id,
  };
  const identity = JSON.stringify(command);
  const current = requested === identity;
  const result = useQuery({
    queryKey: ["formula-authoring", persistence.scope, persistence.accountId, command],
    queryFn: current ? ({ signal }) => persistence.preview(command, signal) : skipToken,
    retry: false,
    gcTime: 0,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });
  return (
    <section aria-label="Formula preview" className="space-y-3 rounded-md border border-subtle p-4">
      <h3 className="text-13 font-medium">Preview formula</h3>
      <Field label="Preview project">
        {(id) => (
          <select
            id={id}
            className={inputStyle}
            value={projectId}
            onChange={(event) => {
              setProjectId(event.target.value);
              setSample(undefined);
            }}
          >
            <option value="">Without a project</option>
            {projects.map((project) => (
              <option key={project.id} value={project.id}>
                {project.name}
              </option>
            ))}
          </select>
        )}
      </Field>
      {projectId && (
        <FormulaSamplePicker
          key={projectId}
          projectId={projectId}
          typeId={typeId}
          persistence={persistence}
          sample={sample}
          onChange={setSample}
        />
      )}
      <Button
        variant="secondary"
        disabled={!typeId || !property.expression.trim() || result.isFetching}
        onClick={() => {
          if (current) void result.refetch();
          else setRequested(identity);
        }}
      >
        Evaluate
      </Button>
      {current && result.isFetching && <output className="block text-12">Evaluating…</output>}
      {current && result.error && (
        <p role="alert" className="text-12 text-danger-primary">
          {typeErrorMessage(result.error)}
        </p>
      )}
      {current && !result.isFetching && !result.error && result.data && <FormulaEvaluation result={result.data} />}
    </section>
  );
}

function FormulaEvaluation({ result }: { result: FormulaAuthoringResult }) {
  if (!result.valid)
    return (
      <p role="alert" className="text-12 text-danger-primary">
        {result.validation_error}
      </p>
    );
  const computed = result.result;
  return (
    <div className="space-y-2 text-12" aria-live="polite">
      <p className="font-medium">Expression is valid.</p>
      {!result.active ? (
        <p>This formula is inactive in the selected context.</p>
      ) : computed?.status === "error" ? (
        <p className="text-danger-primary">Sample calculation: {computed.error}</p>
      ) : (
        <output>{computed?.status === "value" ? String(computed.value) : "No value — an input is empty."}</output>
      )}
      <p className="text-tertiary">Evaluated at {result.evaluated_at} (UTC)</p>
      <details>
        <summary className="cursor-pointer">Effective contexts</summary>
        <ul>
          {result.contexts.map((context) => (
            <li key={context.id}>{context.name}</li>
          ))}
        </ul>
      </details>
    </div>
  );
}
