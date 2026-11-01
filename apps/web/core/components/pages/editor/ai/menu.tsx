/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useRef, useState } from "react";
import type { LucideIcon } from "lucide-react";
import { CornerDownRight, RefreshCcw, Sparkles, TriangleAlert } from "lucide-react";
// plane editor
import type { EditorRefApi } from "@plane/editor";
import { ChevronRightIcon } from "@plane/propel/icons";
// plane ui
import { Tooltip } from "@plane/propel/tooltip";
// components
import { cn } from "@plane/utils/common";
import { RichTextEditor } from "@/components/editor/rich-text";
// plane web constants
import { AI_EDITOR_TASKS, LOADING_TEXTS } from "@plane/constants";
// plane web services
import { useQueryClient } from "@tanstack/react-query";
import { useParams } from "@tanstack/react-router";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useEditorAIResponse } from "./use-editor-ai-response";
import { AskPiMenu } from "./ask-pi-menu";

type Props = {
  editorRef: EditorRefApi | null;
  isOpen: boolean;
  onClose: () => void;
  workspaceId: string;
  workspaceSlug: string;
};

const MENU_ITEMS: {
  icon: LucideIcon;
  key: AI_EDITOR_TASKS;
  label: string;
}[] = [
  {
    key: AI_EDITOR_TASKS.ASK_ANYTHING,
    icon: Sparkles,
    label: "Ask Pi",
  },
];

const TONES_LIST = [
  {
    key: "default",
    label: "Default",
    casual_score: 5,
    formal_score: 5,
  },
  {
    key: "professional",
    label: "💼 Professional",
    casual_score: 0,
    formal_score: 10,
  },
  {
    key: "casual",
    label: "😃 Casual",
    casual_score: 10,
    formal_score: 0,
  },
];

export function EditorAIMenu(props: Props) {
  const accountId = useCurrentUserId();
  const client = useQueryClient();
  const generation = useIdentityGeneration(client);
  const { projectId, pageId } = useParams({ strict: false });
  const scope = JSON.stringify([accountId, generation, props.workspaceSlug, projectId, pageId, props.isOpen]);
  return <EditorAIContent key={scope} {...props} scope={scope} />;
}

function EditorAIContent({ editorRef, onClose, workspaceId, workspaceSlug, scope }: Props & { scope: string }) {
  const [activeTask, setActiveTask] = useState<AI_EDITOR_TASKS | null>(null);
  const { response, isRegenerating, generateResponse } = useEditorAIResponse(workspaceSlug, scope);
  const responseContainerRef = useRef<HTMLDivElement>(null);
  // handle task click
  const handleClick = async (key: AI_EDITOR_TASKS) => {
    const selection = editorRef?.getSelectedText();
    if (!selection || activeTask === key) return;
    setActiveTask(key);
    if (key === AI_EDITOR_TASKS.ASK_ANYTHING) return;
    await generateResponse({ task: key, text_input: selection }, true);
  };
  const scrollToResponse = () => responseContainerRef.current?.scrollTo({ top: 0, behavior: "smooth" });
  const handleRegenerate = async () => {
    const selection = editorRef?.getSelectedText();
    if (!selection || !activeTask) return;
    if (await generateResponse({ task: activeTask, text_input: selection }, false)) scrollToResponse();
  };
  const handleToneChange = async (key: string) => {
    const selectedTone = TONES_LIST.find((tone) => tone.key === key);
    const selection = editorRef?.getSelectedText();
    if (!selectedTone || !selection || !activeTask) return;
    if (
      await generateResponse(
        {
          casual_score: selectedTone.casual_score,
          formal_score: selectedTone.formal_score,
          task: activeTask,
          text_input: selection,
        },
        true
      )
    )
      scrollToResponse();
  };
  // handle replace selected text with the response
  const handleInsertText = (insertOnNextLine: boolean) => {
    if (!response) return;
    editorRef?.insertText(response, insertOnNextLine);
    onClose();
  };

  return (
    <div
      className={cn(
        "flex w-[210px] flex-col rounded-md border-[0.5px] border-strong bg-surface-1 shadow-raised-200 transition-all",
        {
          "w-[700px]": activeTask,
        }
      )}
    >
      <div
        className={cn("flex max-h-72 w-full", {
          "divide-x divide-subtle-1": activeTask,
        })}
      >
        <div className="w-[210px] flex-shrink-0 overflow-y-auto px-2 py-2.5 transition-all">
          {MENU_ITEMS.map((item) => {
            const isActiveTask = activeTask === item.key;

            return (
              <button
                key={item.key}
                type="button"
                className={cn(
                  "flex w-full items-center justify-between gap-2 truncate rounded-sm px-1 py-1.5 text-11 text-secondary transition-colors hover:bg-layer-1",
                  {
                    "bg-layer-1": isActiveTask,
                  }
                )}
                onClick={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  void handleClick(item.key);
                }}
              >
                <span className="flex flex-shrink-0 items-center gap-2 truncate">
                  <item.icon className="size-3 flex-shrink-0" />
                  {item.label}
                </span>
                <ChevronRightIcon
                  className={cn("pointer-events-none size-3 flex-shrink-0 opacity-0 transition-opacity", {
                    "pointer-events-auto opacity-100": isActiveTask,
                  })}
                />
              </button>
            );
          })}
        </div>
        <div
          ref={responseContainerRef}
          className={cn("w-0 flex-shrink-0 overflow-hidden transition-all", {
            "vertical-scrollbar scrollbar-sm w-[490px] overflow-auto": activeTask,
          })}
        >
          {activeTask === AI_EDITOR_TASKS.ASK_ANYTHING ? (
            <AskPiMenu
              handleInsertText={handleInsertText}
              handleRegenerate={handleRegenerate}
              isRegenerating={isRegenerating}
              response={response}
              workspaceSlug={workspaceSlug}
            />
          ) : (
            <>
              <div
                className={cn("flex items-center gap-3 px-4 py-3.5", {
                  "items-start": response,
                })}
              >
                <span className="grid size-7 flex-shrink-0 place-items-center rounded-full border border-subtle text-secondary">
                  <Sparkles className="size-3" />
                </span>
                {response ? (
                  <div>
                    <RichTextEditor
                      displayConfig={{
                        fontSize: "small-font",
                      }}
                      editable={false}
                      id="editor-ai-response"
                      initialValue={response}
                      containerClassName="!p-0 border-none"
                      editorClassName="!pl-0"
                      workspaceId={workspaceId}
                      workspaceSlug={workspaceSlug}
                    />
                    <div className="mt-3 flex items-center gap-4">
                      <button
                        type="button"
                        className="rounded-sm p-1 text-13 font-medium text-tertiary outline-none hover:bg-layer-1"
                        onClick={() => handleInsertText(false)}
                      >
                        Replace selection
                      </button>
                      <Tooltip tooltipContent="Add to next line">
                        <button
                          type="button"
                          className="grid size-6 flex-shrink-0 place-items-center rounded-sm outline-none hover:bg-layer-1"
                          onClick={() => handleInsertText(true)}
                        >
                          <CornerDownRight className="size-4 text-tertiary" />
                        </button>
                      </Tooltip>
                      <Tooltip tooltipContent="Re-generate response">
                        <button
                          type="button"
                          className="grid size-6 flex-shrink-0 place-items-center rounded-sm outline-none hover:bg-layer-1"
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            void handleRegenerate();
                          }}
                          disabled={isRegenerating}
                        >
                          <RefreshCcw
                            className={cn("size-4 text-tertiary", {
                              "animate-spin": isRegenerating,
                            })}
                          />
                        </button>
                      </Tooltip>
                    </div>
                  </div>
                ) : (
                  <p className="text-13 text-secondary">
                    {activeTask ? LOADING_TEXTS[activeTask] : "Pi is writing"}...
                  </p>
                )}
              </div>
              <div className="sticky bottom-0 flex w-full items-center gap-2 bg-surface-1 py-2 pl-[54.8px]">
                {TONES_LIST.map((tone) => (
                  <button
                    key={tone.key}
                    type="button"
                    className={cn(
                      "rounded-sm bg-layer-1 p-1 text-11 font-medium text-secondary transition-colors outline-none",
                      {
                        "bg-accent-primary/20 text-accent-primary": tone.key === "default",
                      }
                    )}
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      void handleToneChange(tone.key);
                    }}
                  >
                    {tone.label}
                  </button>
                ))}
              </div>
            </>
          )}
        </div>
      </div>
      {activeTask && (
        <div className="flex items-center gap-2 rounded-b-md border-t border-subtle bg-surface-2 px-4 py-2 text-tertiary">
          <span className="grid size-4 flex-shrink-0 place-items-center">
            <TriangleAlert className="size-3" />
          </span>
          <p className="flex-shrink-0 text-11 font-medium">
            By using this feature, you consent to sharing the message with a 3rd party service.
          </p>
        </div>
      )}
    </div>
  );
}
