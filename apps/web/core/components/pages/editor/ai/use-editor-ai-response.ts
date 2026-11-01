/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useRef, useState } from "react";
import { setToast, TOAST_TYPE } from "@plane/propel/toast";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { AIService } from "@/services/ai.service";
import type { TTaskPayload } from "@/services/ai.service";

const aiService = new AIService();

export function useEditorAIResponse(workspaceSlug: string, scope: string) {
  const isCurrent = useCommittedScope(scope);
  const request = useRef(0);
  const [response, setResponse] = useState<string>();
  const [isRegenerating, setIsRegenerating] = useState(false);
  const generateResponse = async (payload: TTaskPayload, clearResponse: boolean) => {
    if (!workspaceSlug) return false;
    const attempt = ++request.current;
    if (clearResponse) setResponse(undefined);
    setIsRegenerating(true);
    const ownsResponse = () => isCurrent() && attempt === request.current;
    try {
      const result = await aiService.performEditorTask(workspaceSlug, payload);
      if (!ownsResponse()) return false;
      setResponse(result.response);
      return true;
    } catch {
      if (ownsResponse())
        setToast({ type: TOAST_TYPE.ERROR, title: "Unable to generate a response", message: "Please try again." });
      return false;
    } finally {
      if (ownsResponse()) setIsRegenerating(false);
    }
  };
  return { response, isRegenerating, generateResponse };
}
