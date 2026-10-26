import { createContext, useContext, useState, useMemo } from "react";
import type { Dispatch, ReactNode, SetStateAction } from "react";
import { createPreviewContent } from "./preview-content";
import type { PreviewContent } from "./preview-content";

type PreviewState = {
  content: PreviewContent;
  setContent: Dispatch<SetStateAction<PreviewContent>>;
  notice: string;
  announce: (message: string) => void;
};
const PreviewContext = createContext<PreviewState | null>(null);
export function WorkItemTypesPreviewProvider({ children }: { children: ReactNode }) {
  const [content, setContent] = useState(createPreviewContent);
  const [notice, announce] = useState("");
  const value = useMemo(
    () => ({
      content,
      setContent,
      notice,
      announce,
    }),
    [content, notice]
  );
  return <PreviewContext.Provider value={value}>{children}</PreviewContext.Provider>;
}
export function useTypePreview() {
  const context = useContext(PreviewContext);
  if (!context) throw new Error("Work item type preview requires its provider.");
  return context;
}
