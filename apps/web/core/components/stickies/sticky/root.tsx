/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { Minimize2 } from "lucide-react";
import { useForm, useStore } from "@tanstack/react-form";
// plane types
import type { TSticky } from "@plane/types";
// plane utils
import { cn } from "@plane/utils/common";
// hooks
import { useAutosaveFormSubmit } from "@/hooks/use-autosave-form-submit";
import { useStickies } from "@/components/stickies/provider";
// components
import { STICKY_COLORS_LIST } from "../../editor/sticky-editor/color-palette";
import { StickyDeleteModal } from "../delete-modal";
import { StickyInput } from "./inputs";
import { useStickyOperations } from "./use-operations";

type TProps = {
  onClose?: () => void;
  workspaceSlug: string;
  className?: string;
  stickyId: string;
  showToolbar?: boolean;
  handleLayout?: () => void;
};
export const StickyNote = function StickyNote(props: TProps) {
  const { workspaceSlug, stickyId } = props;
  // store hooks
  const { stickies } = useStickies();
  // sticky operations
  const {
    stickyOperations: { update, remove },
    isCurrent,
    sessionKey,
  } = useStickyOperations({ workspaceSlug, stickyId });
  const stickyData = stickies.get(stickyId);

  if (!stickyData) return null;

  return (
    <StickyNoteForm
      key={JSON.stringify([sessionKey, workspaceSlug, stickyId])}
      {...props}
      stickyData={stickyData}
      update={update}
      remove={remove}
      isCurrent={isCurrent}
    />
  );
};

type StickyNoteFormProps = TProps & {
  stickyData: TSticky;
  isCurrent: () => boolean;
  update: (stickyId: string, data: Partial<TSticky>) => Promise<void>;
  remove: (stickyId: string) => Promise<void>;
};

function StickyNoteForm(props: StickyNoteFormProps) {
  const {
    onClose,
    workspaceSlug,
    className = "",
    stickyId,
    showToolbar,
    handleLayout,
    stickyData,
    update,
    remove,
    isCurrent,
  } = props;
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const editRevision = useRef(0);
  const deleting = useRef(false);
  const saveTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const form = useForm({
    defaultValues: { description_html: stickyData.description_html ?? "" },
    onSubmit: async ({ value, formApi }) => {
      if (!isCurrent() || deleting.current) return;
      if (saveTimer.current !== null) clearTimeout(saveTimer.current);
      saveTimer.current = null;
      const submittedRevision = editRevision.current;
      try {
        await update(stickyId, { description_html: value.description_html });
        if (isCurrent() && editRevision.current === submittedRevision)
          formApi.reset({ description_html: value.description_html });
      } catch (error) {
        if (isCurrent() && editRevision.current === submittedRevision) {
          formApi.setErrorMap({ onSubmit: { form: "Could not save this sticky. Try again.", fields: {} } });
        }
        throw error;
      }
    },
  });
  const submitError = useStore(form.store, (state) =>
    typeof state.errorMap.onSubmit === "string" ? state.errorMap.onSubmit : undefined
  );

  const backgroundColor =
    STICKY_COLORS_LIST.find((c) => c.key === stickyData?.background_color)?.backgroundColor ||
    STICKY_COLORS_LIST[0].backgroundColor;

  const { submit: submitForm, cancelQueued } = useAutosaveFormSubmit(form);
  const scheduleSave = useCallback(() => {
    if (deleting.current) return;
    if (saveTimer.current !== null) clearTimeout(saveTimer.current);
    saveTimer.current = setTimeout(() => {
      saveTimer.current = null;
      submitForm();
    }, 500);
  }, [submitForm]);
  const flushSave = useCallback(() => {
    if (saveTimer.current === null) return;
    clearTimeout(saveTimer.current);
    saveTimer.current = null;
    if (!deleting.current) submitForm();
  }, [submitForm]);
  const cancelSave = useCallback(() => {
    if (saveTimer.current !== null) clearTimeout(saveTimer.current);
    saveTimer.current = null;
  }, []);
  useEffect(() => () => flushSave(), [flushSave]);

  useEffect(() => {
    const descriptionHtml = stickyData.description_html ?? "";
    if (!form.getFieldMeta("description_html")?.isDirty) {
      form.setFieldValue("description_html", descriptionHtml, { dontUpdateMeta: true });
    }
  }, [form, stickyData.description_html]);

  const handleDescriptionChange = useCallback(
    (value: string, fieldChange: (value: string) => void) => {
      if (!isCurrent() || deleting.current) return;
      editRevision.current += 1;
      form.setErrorMap({ onSubmit: undefined });
      fieldChange(value);
    },
    [form, isCurrent]
  );

  const handleColorChange = useCallback(
    async (payload: Partial<TSticky>) => {
      try {
        await update(stickyId, payload);
      } catch {
        // The update operation already reports failures to the user.
      }
    },
    [stickyId, update]
  );

  const handleDelete = async () => {
    if (!stickyId || !isCurrent() || deleting.current) return;
    deleting.current = true;
    cancelSave();
    cancelQueued();
    try {
      await remove(stickyId);
      if (!isCurrent()) return;
      setIsDeleteModalOpen(false);
      onClose?.();
    } catch (error) {
      deleting.current = false;
      // A rejected delete keeps the mounted draft and resumes its cancelled autosave.
      if (isCurrent() && form.state.isDirty) submitForm();
      throw error;
    }
  };

  return (
    <>
      <StickyDeleteModal
        isOpen={isDeleteModalOpen}
        handleSubmit={handleDelete}
        handleClose={() => setIsDeleteModalOpen(false)}
      />
      <form
        onSubmit={(event) => {
          event.preventDefault();
          submitForm();
        }}
      >
        <div
          className={cn("group/sticky flex h-fit w-full flex-col overflow-y-scroll rounded-sm", className)}
          style={{
            backgroundColor,
          }}
        >
          {onClose && (
            <button type="button" className="flex flex-shrink-0 justify-end p-2.5" onClick={onClose}>
              <Minimize2 className="size-4" />
            </button>
          )}
          {/* inputs */}
          <div className="-mt-2">
            <form.Field name="description_html">
              {(field) => (
                <StickyInput
                  workspaceSlug={workspaceSlug}
                  descriptionHtml={field.state.value}
                  handleDescriptionChange={(value) => handleDescriptionChange(value, field.handleChange)}
                  handleUpdate={() => {
                    handleLayout?.();
                    scheduleSave();
                  }}
                  stickyId={stickyId}
                  handleDelete={() => setIsDeleteModalOpen(true)}
                  handleChange={handleColorChange}
                  showToolbar={showToolbar}
                />
              )}
            </form.Field>
          </div>
        </div>
        {typeof submitError === "string" && (
          <div role="alert" className="flex items-center justify-between gap-2 px-4 py-2 text-12 text-danger-primary">
            <span>{submitError}</span>
            <button type="button" className="underline" onClick={submitForm}>
              Retry
            </button>
          </div>
        )}
      </form>
    </>
  );
}
