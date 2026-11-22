import { useLayoutEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import type { IState } from "@plane/types";
import { COLLECTION_OPERATOR, FILTER_FIELD_TYPE, FILTER_NODE_TYPE } from "@plane/types";
import { getStateFilterConfig } from "@plane/utils/work-item-filters";
import { MultiSelectFilterValueInput } from "../../core/components/rich-filters/filter-value-input/select/multi";

const initialStates = [
  { id: "backlog", name: "Backlog" },
  { id: "todo", name: "Todo" },
  { id: "started", name: "In progress" },
].map<IState>(({ id, name }, order) => ({
  id,
  name,
  color: "#64748b",
  default: order === 0,
  description: "",
  group: "unstarted",
  project_id: "fixture",
  workspace_id: "fixture",
  sequence: order,
  order,
}));
const selected = ["backlog", "todo", "started"];
const allowedOperators = new Set([COLLECTION_OPERATOR.IN]);

type Sample = { at: number; stage: string; labels: string; disabled: boolean };
const samples: Sample[] = [];
const host = document.createElement("div");
document.body.append(host);

function sample() {
  const view = host.querySelector<HTMLElement>("[data-stage]");
  const button = view?.querySelector("button");
  const stage = view?.dataset.stage;
  if (!stage || !button) return;
  const labels = button.textContent ?? "";
  const previous = samples.at(-1);
  if (previous?.stage === stage && previous.labels === labels && previous.disabled === button.disabled) return;
  samples.push({ at: Math.round(performance.now()), stage, labels, disabled: button.disabled });
}

new MutationObserver(sample).observe(host, { attributes: true, childList: true, characterData: true, subtree: true });

function Probe() {
  const [states, setStates] = useState(initialStates);
  const [revision, setRevision] = useState(0);
  const config = getStateFilterConfig("state_id")({ isEnabled: true, allowedOperators, states });
  const field = config.supportedOperatorConfigsMap.get(COLLECTION_OPERATOR.IN);

  useLayoutEffect(() => {
    sample();
    Object.assign(window, {
      refreshPreferences: async () => {
        const response = await fetch("/preferences");
        if (!response.ok) throw new Error("Fixture preferences request failed");
        setRevision((current) => current + 1);
      },
      refreshStates: async () => {
        const response = await fetch("/states");
        if (!response.ok) throw new Error("Fixture states request failed");
        const name = await response.text();
        setStates((current) => current.map((state) => (state.id === "todo" ? { ...state, name } : state)));
      },
    });
  });

  if (field?.type !== FILTER_FIELD_TYPE.MULTI_SELECT) throw new Error("State filter did not configure a multi-select");

  const stage = revision > 0 ? "preferences response" : states === initialStates ? "initial" : "states response";
  return (
    <div data-stage={stage}>
      <MultiSelectFilterValueInput
        config={field}
        condition={{
          type: FILTER_NODE_TYPE.CONDITION,
          id: "status",
          property: "state_id",
          operator: COLLECTION_OPERATOR.IN,
          value: selected,
        }}
        onChange={() => {}}
      />
    </div>
  );
}

Object.assign(window, { samples });
createRoot(host).render(<Probe />);
