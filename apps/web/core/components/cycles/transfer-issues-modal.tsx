/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRef } from "react";

import { AlertCircle } from "lucide-react";
import { SearchIcon, CycleIcon, TransferIcon, CloseIcon } from "@plane/propel/icons";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { EIssuesStoreType } from "@plane/types";
import { EModalPosition, EModalWidth, ModalForm, FormErrors } from "@plane/ui";
import { useProjectCycles } from "@/hooks/use-cycles";
import { cycleTransferOptions } from "@/queries/cycle-transfers";
import { cycleIsCompleted } from "@/queries/cycle-module-selectors";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";
import { useIssueListMutationOwner } from "@/hooks/use-issue-list-mutation-owner";
import { captureIssueListMutationOwner, refreshCapturedIssueLists } from "@/queries/issue-list-mutations";

type Props = {
  isOpen: boolean;
  handleClose: () => void;
  cycleId: string;
  workspaceSlug: string;
  projectId: string;
};

export function TransferIssuesModal(props: Props) {
  const { workspaceSlug, projectId } = props;
  return props.isOpen ? (
    <TransferIssuesForm
      key={`${workspaceSlug}/${projectId}/${props.cycleId}`}
      {...props}
      workspaceSlug={workspaceSlug}
      projectId={projectId}
    />
  ) : null;
}

function TransferIssuesForm({ handleClose, cycleId, workspaceSlug, projectId }: Props) {
  // states
  const [query, setQuery] = useState("");

  // store hooks
  const catalogue = useProjectCycles(workspaceSlug, projectId),
    client = useQueryClient();
  const current = useCommittedScope(JSON.stringify([catalogue.scope, cycleId]));
  const resolveListOwner = useIssueListMutationOwner(EIssuesStoreType.CYCLE);

  const [error, setError] = useState("");
  const transfer = useMutation(cycleTransferOptions(client, catalogue.scope, cycleId));
  const acceptedTransfer = useRef<
    | {
        result: NonNullable<typeof transfer.data>;
        capturedLists: ReturnType<typeof captureIssueListMutationOwner>;
      }
    | undefined
  >(undefined);
  const submitTransfer = async (targetCycleId: string) => {
    if (acceptedTransfer.current) return acceptedTransfer.current;
    try {
      const owner = await resolveListOwner(workspaceSlug, projectId);
      if (!owner) throw new Error("Cycle issue access is unavailable. Retry the operation.");
      const capturedLists = captureIssueListMutationOwner(client, owner);
      const result = await transfer.mutateAsync(targetCycleId);
      acceptedTransfer.current = { result, capturedLists };
      return acceptedTransfer.current;
    } catch (cause) {
      if (current()) setError("Unable to transfer work items. Please try again.");
      throw cause;
    }
  };
  const refreshTransfer = async (accepted: NonNullable<typeof acceptedTransfer.current>) => {
    try {
      const { result, capturedLists } = accepted;
      if (!result.captured.isCurrent() || !current()) return;
      await Promise.all([result.refresh(), refreshCapturedIssueLists(client, capturedLists)]);
      if (!current() || !result.captured.isCurrent()) return;
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: "Work items have been transferred successfully",
      });
      handleClose();
    } catch {
      if (current())
        setError("Work items were transferred, but cycle details could not be refreshed. Retry to refresh them.");
      // The transfer is durable; this is a refresh-only retry, not a failed transfer.
    }
  };
  const form = useForm({
    defaultValues: { targetCycleId: "" },
    onSubmit: async ({ value }) => {
      setError("");
      const result = await submitTransfer(value.targetCycleId);
      await refreshTransfer(result);
    },
  });
  const busy = useStore(form.store, (state) => state.isSubmitting);

  const filteredOptions = catalogue.cycles?.filter(
    (cycle) =>
      cycle.id !== cycleId && !cycleIsCompleted(cycle) && cycle.name.toLowerCase().includes(query.toLowerCase())
  );

  return (
    <ModalForm
      busy={busy}
      onSubmit={() => {
        form.setErrorMap({ onSubmit: undefined });
        return form.handleSubmit().catch(() => {});
      }}
      handleClose={handleClose}
      position={EModalPosition.TOP}
      width={EModalWidth.XXL}
    >
      <div className="flex flex-col gap-4 py-5">
        <div className="flex items-center justify-between px-5">
          <div className="flex items-center gap-1">
            <TransferIcon className="w-5 fill-primary" />
            <h4 className="text-18 font-medium text-primary">Transfer work items</h4>
          </div>
          <button type="button" disabled={busy} onClick={handleClose} aria-label="Close transfer">
            <CloseIcon className="h-4 w-4" />
          </button>
        </div>
        <fieldset disabled={busy || transfer.isSuccess}>
          <div className="flex items-center gap-2 border-b border-subtle px-5 pb-3">
            <SearchIcon className="h-4 w-4 text-secondary" />
            <input
              className="text-13 outline-none"
              placeholder="Search for a cycle..."
              onChange={(e) => setQuery(e.target.value)}
              value={query}
            />
          </div>
          <form.Field
            name="targetCycleId"
            validators={{
              onChange: ({ value }) => (value && value !== cycleId ? undefined : "Select a different cycle."),
            }}
          >
            {(field) => (
              <div className="flex w-full flex-col items-start gap-2 px-5">
                {(catalogue.isError || catalogue.isAccessDenied) && (
                  <CatalogueReadFailure label="cycles" retry={catalogue.retry} />
                )}
                {filteredOptions ? (
                  filteredOptions.length > 0 ? (
                    filteredOptions.map((cycleDetails) => {
                      const optionId = cycleDetails.id;

                      if (!cycleDetails) return null;

                      return (
                        <button
                          key={optionId}
                          type="button"
                          aria-pressed={field.state.value === optionId}
                          className="flex w-full items-center gap-4 rounded-sm px-4 py-3 text-13 text-secondary hover:bg-surface-2"
                          onClick={() => {
                            field.handleChange(optionId);
                            form.setErrorMap({ onSubmit: undefined });
                            void form.handleSubmit().catch(() => {});
                          }}
                        >
                          <CycleIcon className="h-5 w-5" />
                          <div className="flex w-full justify-between truncate">
                            <span className="truncate">{cycleDetails?.name}</span>
                            {cycleDetails.status && (
                              <span className="flex flex-shrink-0 items-center rounded-full bg-layer-1 px-2 capitalize">
                                {cycleDetails.status.toLocaleLowerCase()}
                              </span>
                            )}
                          </div>
                        </button>
                      );
                    })
                  ) : (
                    <div className="flex w-full items-center justify-center gap-4 p-5 text-13">
                      <AlertCircle className="h-3.5 w-3.5 text-secondary" />
                      <span className="text-center text-secondary">
                        You don’t have any current cycle. Please create one to transfer the work items.
                      </span>
                    </div>
                  )
                ) : (
                  <p className="text-center text-secondary">Loading...</p>
                )}
                <FormErrors errors={field.state.meta.errors} />
              </div>
            )}
          </form.Field>
        </fieldset>
        <div className="px-5">
          <FormErrors errors={[error]} />
          {error && (
            <button type="submit" disabled={busy} className="mt-2 text-13 text-accent-primary">
              {transfer.isSuccess ? "Retry refresh" : "Retry transfer"}
            </button>
          )}
        </div>
      </div>
    </ModalForm>
  );
}
