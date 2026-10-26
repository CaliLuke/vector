/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useForm, useStore } from "@tanstack/react-form";
import { TextSelection } from "@tiptap/pm/state";
import { Link2Off } from "lucide-react";
import { useCallback, useId } from "react";
// components
import type { LinkViewProps } from "@/components/links";
// helpers
import { isValidHttpUrl } from "@/helpers/common";

type LinkEditViewProps = {
  viewProps: LinkViewProps;
};

export function LinkEditView({ viewProps }: LinkEditViewProps) {
  const { editor, from, to } = viewProps;
  return <LinkEditForm key={`${editor.instanceId}:${from}:${to}`} viewProps={viewProps} />;
}

function LinkEditForm({ viewProps }: LinkEditViewProps) {
  const { editor, from, to, url: initialUrl, text: initialText, closeLinkView } = viewProps;
  const originalText = initialText ?? "";
  const id = useId();

  const getTarget = useCallback(() => {
    if (from < 0 || to <= from || to > editor.state.doc.content.size) return undefined;
    const node = editor.state.doc.nodeAt(from);
    const linkMark = node?.marks.find((mark) => mark.type === editor.schema.marks.link);
    if (
      !node?.isText ||
      !linkMark ||
      node.text !== originalText ||
      from + node.nodeSize !== to ||
      linkMark.attrs.href !== initialUrl
    ) {
      return undefined;
    }
    return { node, linkMark };
  }, [editor, from, to, originalText, initialUrl]);

  const getEditError = useCallback(
    (url: string) => {
      if (!isValidHttpUrl(url).isValid) return "Enter a valid URL";
      if (!getTarget()) return "This link changed. Close the editor and try again.";
      return undefined;
    },
    [getTarget]
  );

  const form = useForm({
    defaultValues: { url: initialUrl, text: originalText },
    validators: {
      onSubmit: ({ value }): string | undefined => getEditError(value.url),
    },
    onSubmit: ({ value }) => {
      const error = getEditError(value.url);
      if (error) throw new Error(error);

      const target = getTarget();
      if (!target) throw new Error("This link changed. Close the editor and try again.");

      const updatedLink = editor.schema.marks.link.create({
        ...target.linkMark.attrs,
        href: isValidHttpUrl(value.url).url,
      });
      let transaction = editor.state.tr;
      if (value.text === originalText) {
        transaction = transaction.removeMark(from, to, editor.schema.marks.link).addMark(from, to, updatedLink);
      } else {
        const marks = [...target.node.marks.filter((mark) => mark.type !== editor.schema.marks.link), updatedLink];
        transaction = transaction.replaceWith(from, to, editor.schema.text(value.text, marks));
        transaction = transaction.setSelection(TextSelection.create(transaction.doc, from, from + value.text.length));
      }
      editor.view.dispatch(transaction);
      closeLinkView();
    },
  });

  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const formError = useStore(form.store, (state) => state.errorMap.onSubmit);

  const removeLink = useCallback(() => {
    const target = getTarget();
    if (!target) {
      form.setErrorMap({ onSubmit: "This link changed. Close the editor and try again." });
      return;
    }
    editor.view.dispatch(editor.state.tr.removeMark(from, to, target.linkMark.type));
    closeLinkView();
  }, [editor, from, to, closeLinkView, form, getTarget]);

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        event.stopPropagation();
        form.setErrorMap({ onSubmit: undefined });
        void form.handleSubmit().catch((error: unknown) => {
          form.setErrorMap({
            onSubmit: error instanceof Error ? error.message : "Could not update this link",
          });
        });
      }}
      noValidate
      className="shadow-md animate-in fade-in flex translate-y-1 flex-col gap-3 rounded-sm border-2 border-subtle bg-layer-1 p-2"
      style={{ transition: "all 0.1s cubic-bezier(.55, .085, .68, .53)" }}
      tabIndex={0}
    >
      <form.Field name="url">
        {(field) => (
          <div className="flex flex-col gap-1">
            <label htmlFor={`${id}-url`} className="inline-block text-11 font-semibold text-placeholder">
              URL
            </label>
            <input
              id={`${id}-url`}
              name={field.name}
              placeholder="Enter or paste URL"
              onClick={(event) => event.stopPropagation()}
              className="w-[280px] rounded-md border border-strong bg-layer-1 p-2 text-13 text-primary outline-none"
              value={field.state.value}
              onChange={(event) => {
                form.setErrorMap({ onSubmit: undefined });
                field.handleChange(event.target.value);
              }}
              onBlur={field.handleBlur}
              autoFocus
              disabled={isSubmitting}
              aria-invalid={Boolean(formError)}
            />
          </div>
        )}
      </form.Field>
      <form.Field name="text">
        {(field) => (
          <div className="flex flex-col gap-1">
            <label htmlFor={`${id}-text`} className="inline-block text-11 font-semibold text-placeholder">
              Text
            </label>
            <input
              id={`${id}-text`}
              name={field.name}
              placeholder="Enter Text to display"
              onClick={(event) => event.stopPropagation()}
              className="w-[280px] rounded-md border border-strong bg-layer-1 p-2 text-13 text-primary outline-none"
              value={field.state.value}
              onChange={(event) => {
                if (event.target.value.trim() !== "") field.handleChange(event.target.value);
              }}
              onBlur={field.handleBlur}
              disabled={isSubmitting}
            />
          </div>
        )}
      </form.Field>
      {formError && (
        <p role="alert" className="text-11 text-danger-primary">
          {formError}
        </p>
      )}
      <div className="bg-strong mb-1 h-[1px] w-full gap-2" />
      <div className="flex items-center justify-between gap-2 text-13 text-secondary">
        <Link2Off size={14} className="inline-block" />
        <button
          type="button"
          onClick={removeLink}
          disabled={isSubmitting}
          className="cursor-pointer transition-colors hover:text-placeholder"
        >
          Remove Link
        </button>
        <button
          type="submit"
          disabled={isSubmitting}
          className="cursor-pointer rounded-sm bg-accent-primary px-2 py-1 text-primary transition-colors hover:bg-accent-primary/80"
        >
          Save Link
        </button>
      </div>
    </form>
  );
}
