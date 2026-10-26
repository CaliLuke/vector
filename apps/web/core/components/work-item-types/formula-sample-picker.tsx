import { useState } from "react";
import { useProjectIssueSearch } from "@/hooks/use-project-issue-search";
import useDebounce from "@/hooks/use-debounce";
import { typeErrorMessage } from "@/services/work-item-types.service";
import type { FormulaPersistence } from "./formula-editor";
import { Field, inputStyle } from "./controls";

export function FormulaSamplePicker({
  projectId,
  typeId,
  persistence,
  sample,
  onChange,
}: {
  projectId: string;
  typeId: string;
  persistence: FormulaPersistence;
  sample?: { id: string; name: string };
  onChange: (sample: { id: string; name: string } | undefined) => void;
}) {
  const [search, setSearch] = useState("");
  const debouncedSearch = useDebounce(search, 300);
  const samples = useProjectIssueSearch(
    persistence.scope,
    projectId,
    {
      search: debouncedSearch,
      workspace_search: false,
      type_id: typeId,
    },
    debouncedSearch.trim().length >= 2
  );
  const choices = (samples.data ?? []).filter((item) => item.type_id === typeId);
  const selectedInResults = choices.some((item) => item.id === sample?.id);
  return (
    <>
      <Field label="Find a sample work item">
        {(id) => (
          <input
            id={id}
            className={inputStyle}
            value={search}
            placeholder="Search by title or identifier"
            onChange={(event) => setSearch(event.target.value)}
          />
        )}
      </Field>
      <Field label="Sample work item">
        {(id) => (
          <select
            id={id}
            className={inputStyle}
            value={sample?.id ?? ""}
            onChange={(event) => {
              const item = choices.find((choice) => choice.id === event.target.value);
              onChange(item ? { id: item.id, name: item.name } : undefined);
            }}
          >
            <option value="">New work item · defaults only</option>
            {sample && !selectedInResults && <option value={sample.id}>{sample.name}</option>}
            {choices.map((item) => (
              <option key={item.id} value={item.id}>
                {item.project__identifier}-{item.sequence_id} · {item.name}
              </option>
            ))}
          </select>
        )}
      </Field>
      {samples.isFetching && <output className="text-12">Searching work items…</output>}
      {samples.error && (
        <p role="alert" className="text-12 text-danger-primary">
          {typeErrorMessage(samples.error)}
        </p>
      )}
      {!samples.isFetching && !samples.error && debouncedSearch.trim().length >= 2 && !choices.length && (
        <p className="text-12 text-secondary">No matching work items of this type. Refine your search.</p>
      )}
    </>
  );
}
