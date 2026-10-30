/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useEffect, useState, useRef } from "react";
import type { Placement } from "@popperjs/core";
import { useForm, useStore } from "@tanstack/react-form";
import { usePopper } from "react-popper";
import { AlertCircle } from "lucide-react";
import { Popover, Transition } from "@headlessui/react";
// plane imports
import type { EditorRefApi } from "@plane/editor";
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { Input } from "@plane/ui";
// components
import { RichTextEditor } from "@/components/editor/rich-text";
// services
import useKeypress from "@/hooks/use-keypress";
import { getAiErrorMessage } from "@/helpers/ai-error.helper";
import { AIService } from "@/services/ai.service";
const aiService = new AIService();

type Props = {
  isOpen: boolean;
  handleClose: () => void;
  onResponse: (response: string) => void;
  onError?: (error: unknown) => void;
  placement?: Placement;
  prompt?: string;
  button: React.ReactNode;
  onOpen: () => void;
  disabled?: boolean;
  className?: string;
  workspaceId: string;
  workspaceSlug: string;
  projectId: string;
};

export function GptAssistantPopover(props: Props) {
  const {
    isOpen,
    handleClose,
    onResponse,
    onError,
    placement,
    prompt,
    button,
    onOpen,
    disabled,
    className = "",
    workspaceId,
    workspaceSlug,
    projectId,
  } = props;
  // states
  const [response, setResponse] = useState("");
  const [invalidResponse, setInvalidResponse] = useState(false);
  const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);
  const [popperElement, setPopperElement] = useState<HTMLElement | null>(null);
  // refs
  const editorRef = useRef<EditorRefApi>(null);
  const responseRef = useRef<EditorRefApi>(null);
  // popper
  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: placement ?? "auto",
  });
  const taskInput = useRef<HTMLInputElement>(null);
  const form = useForm({
    defaultValues: { task: "" },
    validators: {
      onSubmit: ({ value }) => (value.task.trim() ? undefined : "Please enter some task to get AI assistance."),
    },
    onSubmit: async ({ value }) => {
      await callAIService(value);
    },
  });
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const formError = useStore(form.store, (state) => state.errorMap.onSubmit);

  const onClose = () => {
    if (isSubmitting) return;
    handleClose();
    setResponse("");
    setInvalidResponse(false);
    form.reset();
  };

  const handleServiceError = (error: unknown) => {
    setToast({ type: TOAST_TYPE.ERROR, title: "Error!", message: getAiErrorMessage(error) });
    onError?.(error);
  };

  const callAIService = async (formData: { task: string }) => {
    try {
      const res = await aiService.createGptTask(workspaceSlug, {
        prompt: prompt || "",
        task: formData.task,
      });

      setResponse(res.response_html);
      taskInput.current?.focus();

      setInvalidResponse(res.response === "");
    } catch (err) {
      handleServiceError(err);
      throw err;
    }
  };

  useEffect(() => {
    if (isOpen) taskInput.current?.focus();
  }, [isOpen]);

  useEffect(() => {
    editorRef.current?.setEditorValue(prompt || "");
  }, [editorRef, prompt]);

  useEffect(() => {
    responseRef.current?.setEditorValue(`<p>${response}</p>`);
  }, [response, responseRef]);

  useKeypress("Escape", () => {
    if (isOpen) onClose();
  });

  const responseActionButton = response !== "" && (
    <Button
      type="button"
      disabled={isSubmitting}
      variant="primary"
      onClick={() => {
        onResponse(response);
        onClose();
      }}
    >
      Use this response
    </Button>
  );

  const generateResponseButtonText = isSubmitting
    ? "Generating response..."
    : response === ""
      ? "Generate response"
      : "Generate again";

  return (
    <Popover as="div" className={`relative w-min text-left`}>
      <Popover.Button
        type="button"
        ref={setReferenceElement}
        className="flex items-center gap-1 rounded-sm bg-surface-2 px-1.5 py-1 text-caption-sm-regular hover:bg-layer-1"
        disabled={disabled}
        onClick={isOpen ? onClose : onOpen}
      >
        {button}
      </Popover.Button>
      <Transition
        show={isOpen}
        as={React.Fragment}
        enter="transition ease-out duration-100"
        enterFrom="transform opacity-0 scale-95"
        enterTo="transform opacity-100 scale-100"
        leave="transition ease-in duration-75"
        leaveFrom="transform opacity-100 scale-100"
        leaveTo="transform opacity-0 scale-95"
      >
        <Popover.Panel
          as="div"
          className={`shadow fixed z-10 flex w-full max-w-full min-w-[50rem] flex-col space-y-4 overflow-hidden rounded-[10px] border border-subtle bg-surface-1 p-4 ${className}`}
          ref={setPopperElement}
          style={styles.popper}
          {...attributes.popper}
        >
          <div className="vertical-scroll-enable max-h-72 space-y-4 overflow-y-auto">
            {prompt && (
              <div className="text-13">
                Content:
                <RichTextEditor
                  editable={false}
                  id="ai-assistant-content"
                  initialValue={prompt}
                  containerClassName="-m-3"
                  ref={editorRef}
                  workspaceId={workspaceId}
                  workspaceSlug={workspaceSlug}
                  projectId={projectId}
                />
              </div>
            )}
            {response !== "" && (
              <div className="page-block-section max-h-[8rem] text-13">
                Response:
                <RichTextEditor
                  editable={false}
                  id="ai-assistant-response"
                  initialValue={`<p>${response}</p>`}
                  ref={responseRef}
                  workspaceId={workspaceId}
                  workspaceSlug={workspaceSlug}
                  projectId={projectId}
                />
              </div>
            )}
            {invalidResponse && (
              <div className="text-13 text-danger-primary">
                No response could be generated. This may be due to insufficient content or task information. Please try
                again.
              </div>
            )}
          </div>
          <form.Field name="task">
            {(field) => (
              <Input
                onKeyDown={(event) => {
                  if (event.key === "Enter" && !event.shiftKey) {
                    event.preventDefault();
                    event.stopPropagation();
                    form.setErrorMap({ onSubmit: undefined });
                    void form.handleSubmit().catch(() => {});
                  }
                }}
                id="task"
                name={field.name}
                type="text"
                value={field.state.value}
                onChange={(event) => field.handleChange(event.target.value)}
                onBlur={field.handleBlur}
                ref={taskInput}
                disabled={isSubmitting}
                placeholder={prompt ? "Tell AI what action to perform on this content..." : "Ask AI anything..."}
                className="w-full"
              />
            )}
          </form.Field>
          {formError && (
            <p role="alert" className="text-13 text-danger-primary">
              {formError}
            </p>
          )}
          <div className="flex justify-between gap-2">
            {responseActionButton ? (
              <>{responseActionButton}</>
            ) : (
              <>
                <div className="flex items-start justify-center gap-2 text-13 text-accent-primary">
                  <AlertCircle className="h-4 w-4" />
                  <p>By using this feature, you consent to sharing the message with a 3rd party service. </p>
                </div>
              </>
            )}
            <div className="flex items-center gap-2">
              <Button type="button" variant="secondary" disabled={isSubmitting} onClick={onClose}>
                Close
              </Button>
              <Button
                type="button"
                variant="primary"
                onClick={() => {
                  form.setErrorMap({ onSubmit: undefined });
                  void form.handleSubmit().catch(() => {});
                }}
                loading={isSubmitting}
              >
                {generateResponseButtonText}
              </Button>
            </div>
          </div>
        </Popover.Panel>
      </Transition>
    </Popover>
  );
}
