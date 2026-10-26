/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/core";

import { useForm, useStore } from "@tanstack/react-form";
import { LinkIcon, TrashIcon, CheckIcon } from "@plane/propel/icons";
// plane imports
import { cn } from "@plane/utils/common";
// constants
import { CORE_EXTENSIONS } from "@/constants/extension";
// helpers
import { isValidHttpUrl } from "@/helpers/common";
import { setLinkEditor, unsetLinkEditor } from "@/helpers/editor-commands";
import { FloatingMenuRoot } from "../floating-menu/root";
import { useFloatingMenu } from "../floating-menu/use-floating-menu";

type Props = {
  editor: Editor;
};

const getUrlValidationError = (url: string): string | undefined => {
  if (!url.trim() || !isValidHttpUrl(url).isValid) return "Please enter a valid URL";
  return undefined;
};

export function BubbleMenuLinkSelector(props: Props) {
  const { editor } = props;
  const { options, getReferenceProps, getFloatingProps } = useFloatingMenu({});
  const { context } = options;
  const linkHref = editor.getAttributes("link").href || "";

  return (
    <FloatingMenuRoot
      classNames={{
        buttonContainer: "h-full",
        button: cn(
          "flex h-full items-center gap-1 rounded-sm px-3 text-13 font-medium whitespace-nowrap text-tertiary transition-colors hover:bg-layer-1 active:bg-layer-1",
          {
            "bg-layer-1": context.open,
            "text-primary": editor.isActive(CORE_EXTENSIONS.CUSTOM_LINK),
          }
        ),
      }}
      getFloatingProps={getFloatingProps}
      getReferenceProps={getReferenceProps}
      menuButton={
        <>
          Link
          <LinkIcon className="size-3 shrink-0" />
        </>
      }
      options={options}
    >
      <LinkUrlForm
        key={editor.instanceId}
        editor={editor}
        linkHref={linkHref}
        onClose={() => context.onOpenChange(false)}
      />
    </FloatingMenuRoot>
  );
}

function LinkUrlForm({ editor, linkHref, onClose }: { editor: Editor; linkHref: string; onClose: () => void }) {
  const form = useForm({
    defaultValues: { url: linkHref },
    onSubmit: async ({ value }) => {
      const { isValid, url: validatedUrl } = isValidHttpUrl(value.url);
      if (!isValid) throw new Error("Cannot apply an invalid link URL");
      setLinkEditor(editor, validatedUrl);
      onClose();
    },
  });
  const busy = useStore(form.store, (state) => state.isSubmitting);

  return (
    <form.Field
      name="url"
      validators={{
        onSubmit: ({ value }) => getUrlValidationError(value),
      }}
    >
      {(field) => {
        const clearError = () => field.setErrorMap({ onSubmit: undefined });
        const error = field.state.meta.errors[0];
        return (
          <form
            className="mt-1 w-60 rounded-md bg-surface-1 shadow-raised-200"
            noValidate
            onSubmit={(event) => {
              event.preventDefault();
              event.stopPropagation();
              void form.handleSubmit().catch(() => {
                field.setErrorMap({ onSubmit: "Could not apply the link. Please try again." });
              });
            }}
          >
            <div
              className={cn("flex rounded-sm border-[0.5px] border-strong transition-colors", {
                "border-danger-strong": Boolean(error),
              })}
            >
              <input
                name={field.name}
                aria-label="Link URL"
                aria-invalid={Boolean(error)}
                type="url"
                placeholder="Enter or paste a link"
                value={field.state.value}
                onChange={(event) => {
                  clearError();
                  field.handleChange(event.target.value);
                }}
                onBlur={field.handleBlur}
                onClick={(event) => event.stopPropagation()}
                className="flex-1 rounded-sm border-r-[0.5px] border-strong bg-surface-1 px-1.5 py-2 text-11 outline-none placeholder:text-placeholder"
                onKeyDown={(event) => {
                  clearError();
                  if (event.key === "Enter") {
                    event.preventDefault();
                    event.currentTarget.form?.requestSubmit();
                  }
                }}
                autoFocus
                disabled={busy}
              />
              {linkHref ? (
                <button
                  type="button"
                  aria-label="Remove link"
                  disabled={busy}
                  className="grid place-items-center rounded-xs p-1 text-danger-primary transition-all hover:bg-danger-subtle-hover"
                  onClick={(event) => {
                    unsetLinkEditor(editor);
                    event.stopPropagation();
                    onClose();
                  }}
                >
                  <TrashIcon className="size-4" />
                </button>
              ) : (
                <button
                  type="submit"
                  aria-label="Apply link"
                  disabled={busy}
                  className="grid aspect-square h-full place-items-center rounded-xs p-1 text-tertiary transition-all hover:bg-layer-1"
                  onClick={(event) => event.stopPropagation()}
                >
                  <CheckIcon className="size-4" />
                </button>
              )}
            </div>
            {error && (
              <p
                role="alert"
                className="animate-in fade-in slide-in-from-top-0 pointer-events-none my-1 px-2 text-11 text-danger-primary"
              >
                {error}
              </p>
            )}
          </form>
        );
      }}
    </form.Field>
  );
}
