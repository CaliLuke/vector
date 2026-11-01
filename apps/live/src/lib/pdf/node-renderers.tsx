/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Image, Link, Text, View } from "@react-pdf/renderer";
import type { Style } from "@react-pdf/types";
import type { ReactElement } from "react";
import { CORE_EXTENSIONS } from "@plane/utils/editor/common";
import { BACKGROUND_COLORS, EDITOR_BACKGROUND_COLORS, resolveColorForPdf, TEXT_COLORS } from "./colors";
import { CheckIcon, ClipboardIcon, DocumentIcon, GlobeIcon, LightbulbIcon, LinkIcon } from "./icons";
import { applyMarks } from "./mark-renderers";
import { pdfStyles } from "./styles";
import type { KeyGenerator, NodeRendererRegistry, PDFExportMetadata, PDFRenderContext, TipTapNode } from "./types";

const stringAttribute = (value: unknown): string | undefined => (typeof value === "string" ? value : undefined);
const numberAttribute = (value: unknown): number | undefined =>
  typeof value === "number" && Number.isFinite(value) ? value : undefined;
const columnWidth = (value: unknown): number | undefined =>
  Array.isArray(value) ? numberAttribute(value[0]) : undefined;
const headingStyles = [
  pdfStyles.heading1,
  pdfStyles.heading2,
  pdfStyles.heading3,
  pdfStyles.heading4,
  pdfStyles.heading5,
  pdfStyles.heading6,
];
const backgroundColors: Readonly<Record<string, string | undefined>> = EDITOR_BACKGROUND_COLORS;

const calloutIcons: Readonly<Record<string, typeof LightbulbIcon | undefined>> = {
  FileText: DocumentIcon,
  File: DocumentIcon,
  Link: LinkIcon,
  Globe: GlobeIcon,
  Clipboard: ClipboardIcon,
  CheckSquare: CheckIcon,
  Check: CheckIcon,
  Lightbulb: LightbulbIcon,
};

const getCalloutIcon = (node: TipTapNode, color: string): ReactElement => {
  const logoInUse = stringAttribute(node.attrs?.["data-logo-in-use"]);
  const iconName = stringAttribute(node.attrs?.["data-icon-name"]);
  const iconColor = stringAttribute(node.attrs?.["data-icon-color"]) || color;

  if (logoInUse === "emoji") {
    const emojiUnicode = stringAttribute(node.attrs?.["data-emoji-unicode"]);
    if (emojiUnicode) {
      return <Text style={{ fontSize: 14 }}>{emojiUnicode}</Text>;
    }
  }

  if (iconName) {
    const Icon = calloutIcons[iconName] ?? LightbulbIcon;
    return <Icon size={16} color={iconColor} />;
  }

  return <LightbulbIcon size={16} color={color} />;
};

export const createKeyGenerator = (): KeyGenerator => {
  let counter = 0;
  return () => `node-${counter++}`;
};

const renderTextWithMarks = (node: TipTapNode, getKey: KeyGenerator): ReactElement => {
  const style = applyMarks(node.marks, {});
  const hasLink = node.marks?.find((m) => m.type === "link");

  if (hasLink) {
    const href = stringAttribute(hasLink.attrs?.href) || "#";
    return (
      <Link key={getKey()} src={href} style={{ ...pdfStyles.link, ...style }}>
        {node.text || ""}
      </Link>
    );
  }

  return (
    <Text key={getKey()} style={style}>
      {node.text || ""}
    </Text>
  );
};

const getTextAlignStyle = (textAlign: string | null | undefined): Style => {
  switch (textAlign) {
    case "left":
    case "right":
    case "center":
    case "justify":
      return { textAlign };
    default:
      return {};
  }
};

const getFlexAlignStyle = (textAlign: string | null | undefined): Style => {
  if (!textAlign) return {};
  if (textAlign === "right") return { alignItems: "flex-end" };
  if (textAlign === "center") return { alignItems: "center" };
  return {};
};

function renderTableCell(
  node: TipTapNode,
  children: ReactElement[],
  ctx: PDFRenderContext,
  style: Style
): ReactElement {
  const width = columnWidth(node.attrs?.colwidth);
  const background = stringAttribute(node.attrs?.background);
  const resolvedBackground = background ? resolveColorForPdf(background, "background") : null;
  return (
    <View
      key={ctx.getKey()}
      style={[
        style,
        width ? { width, flex: undefined } : {},
        resolvedBackground ? { backgroundColor: resolvedBackground } : {},
      ]}
    >
      {children}
    </View>
  );
}

function imageAlignment(alignment: unknown): Style {
  if (alignment === "center") return { alignItems: "center" };
  if (alignment === "right") return { alignItems: "flex-end" };
  return { alignItems: "flex-start" };
}

function resolvedImageSource(src: string, urls: PDFExportMetadata["resolvedImageUrls"]): string | undefined {
  return urls && Object.hasOwn(urls, src) ? stringAttribute(urls[src]) : undefined;
}

function parsedImageWidth(value: unknown): number | undefined {
  return numberAttribute(typeof value === "string" ? parseInt(value, 10) : value);
}

function renderImage(node: TipTapNode, ctx: PDFRenderContext, resolveAsset: boolean): ReactElement {
  const src = stringAttribute(node.attrs?.src);
  if (ctx.metadata?.noAssets || !src) return <View key={ctx.getKey()} />;
  const rawWidth = node.attrs?.width;
  const width = resolveAsset ? parsedImageWidth(rawWidth) : numberAttribute(rawWidth);
  const alignment = imageAlignment(node.attrs?.alignment);
  const resolvedSrc = (resolveAsset && resolvedImageSource(src, ctx.metadata?.resolvedImageUrls)) || src;
  if (resolveAsset && !resolvedSrc.startsWith("http") && !resolvedSrc.startsWith("data:")) {
    return (
      <View key={ctx.getKey()} style={[pdfStyles.imagePlaceholder, alignment]}>
        <Text style={pdfStyles.imagePlaceholderText}>[Image: {src.slice(0, 8)}...]</Text>
      </View>
    );
  }
  return (
    <View key={ctx.getKey()} style={[{ width: "100%" }, alignment]}>
      <Image
        src={resolvedSrc}
        style={[pdfStyles.image, width ? { width, maxHeight: 500 } : { maxWidth: 400, maxHeight: 500 }]}
      />
    </View>
  );
}

const nodeRenderers: NodeRendererRegistry = {
  doc: (_node: TipTapNode, children: ReactElement[], ctx: PDFRenderContext): ReactElement => (
    <View key={ctx.getKey()}>{children}</View>
  ),

  text: (node: TipTapNode, _children: ReactElement[], ctx: PDFRenderContext): ReactElement =>
    renderTextWithMarks(node, ctx.getKey),

  paragraph: (node: TipTapNode, children: ReactElement[], ctx: PDFRenderContext): ReactElement => {
    const textAlign = stringAttribute(node.attrs?.textAlign);
    const background = stringAttribute(node.attrs?.backgroundColor);
    const alignStyle = getTextAlignStyle(textAlign);
    const flexStyle = getFlexAlignStyle(textAlign);
    const resolvedBgColor =
      background && background !== "default" ? resolveColorForPdf(background, "background") : null;
    const bgStyle = resolvedBgColor ? { backgroundColor: resolvedBgColor } : {};

    return (
      <View key={ctx.getKey()} style={[pdfStyles.paragraphWrapper, flexStyle, bgStyle]}>
        <Text style={[pdfStyles.paragraph, alignStyle, bgStyle]}>{children}</Text>
      </View>
    );
  },

  heading: (node: TipTapNode, children: ReactElement[], ctx: PDFRenderContext): ReactElement => {
    const level = numberAttribute(node.attrs?.level) || 1;
    const style = headingStyles[level - 1] ?? pdfStyles.heading1;
    const textAlign = stringAttribute(node.attrs?.textAlign);
    const alignStyle = getTextAlignStyle(textAlign);
    const flexStyle = getFlexAlignStyle(textAlign);

    return (
      <View key={ctx.getKey()} style={flexStyle}>
        <Text style={[style, alignStyle]}>{children}</Text>
      </View>
    );
  },

  blockquote: (_node: TipTapNode, children: ReactElement[], ctx: PDFRenderContext): ReactElement => (
    <View key={ctx.getKey()} style={pdfStyles.blockquote} wrap={false}>
      {children}
    </View>
  ),

  codeBlock: (node: TipTapNode, _children: ReactElement[], ctx: PDFRenderContext): ReactElement => {
    const codeContent = node.content?.map((c) => c.text || "").join("") || "";
    return (
      <View key={ctx.getKey()} style={pdfStyles.codeBlock} wrap={false}>
        <Text>{codeContent}</Text>
      </View>
    );
  },

  bulletList: (node: TipTapNode, children: ReactElement[], ctx: PDFRenderContext): ReactElement => {
    const nestingLevel = ctx.layout?.nestingLevel || 0;
    const indentStyle = nestingLevel > 0 ? { marginLeft: 18 } : {};
    return (
      <View key={ctx.getKey()} style={[pdfStyles.bulletList, indentStyle]}>
        {children}
      </View>
    );
  },

  orderedList: (node: TipTapNode, children: ReactElement[], ctx: PDFRenderContext): ReactElement => {
    const nestingLevel = ctx.layout?.nestingLevel || 0;
    const indentStyle = nestingLevel > 0 ? { marginLeft: 18 } : {};
    return (
      <View key={ctx.getKey()} style={[pdfStyles.orderedList, indentStyle]}>
        {children}
      </View>
    );
  },

  listItem: (node: TipTapNode, children: ReactElement[], ctx: PDFRenderContext): ReactElement => {
    const isOrdered = ctx.layout?.parentType === "orderedList";
    const index = ctx.layout?.listItemIndex || 0;

    const bullet = isOrdered ? `${index}.` : "•";

    const textAlign = ctx.layout?.textAlign;
    const flexStyle = getFlexAlignStyle(textAlign);

    return (
      <View key={ctx.getKey()} style={[pdfStyles.listItem, flexStyle]} wrap={false}>
        <View style={pdfStyles.listItemBullet}>
          <Text>{bullet}</Text>
        </View>
        <View style={pdfStyles.listItemContent}>{children}</View>
      </View>
    );
  },

  taskList: (_node: TipTapNode, children: ReactElement[], ctx: PDFRenderContext): ReactElement => (
    <View key={ctx.getKey()} style={pdfStyles.taskList}>
      {children}
    </View>
  ),

  taskItem: (node: TipTapNode, children: ReactElement[], ctx: PDFRenderContext): ReactElement => {
    const checked = node.attrs?.checked === true;
    return (
      <View key={ctx.getKey()} style={pdfStyles.taskItem} wrap={false}>
        <View style={checked ? [pdfStyles.taskCheckbox, pdfStyles.taskCheckboxChecked] : pdfStyles.taskCheckbox}>
          {checked && <CheckIcon size={8} color="#ffffff" />}
        </View>
        <View style={pdfStyles.listItemContent}>{children}</View>
      </View>
    );
  },

  table: (_node: TipTapNode, children: ReactElement[], ctx: PDFRenderContext): ReactElement => (
    <View key={ctx.getKey()} style={pdfStyles.table}>
      {children}
    </View>
  ),

  tableRow: (node: TipTapNode, children: ReactElement[], ctx: PDFRenderContext): ReactElement => {
    const isHeader = ctx.layout?.isHeader === true;
    return (
      <View key={ctx.getKey()} style={isHeader ? pdfStyles.tableHeaderRow : pdfStyles.tableRow} wrap={false}>
        {children}
      </View>
    );
  },

  tableHeader: (node, children, ctx) => renderTableCell(node, children, ctx, pdfStyles.tableHeaderCell),
  tableCell: (node, children, ctx) => renderTableCell(node, children, ctx, pdfStyles.tableCell),

  horizontalRule: (_node: TipTapNode, _children: ReactElement[], ctx: PDFRenderContext): ReactElement => (
    <View key={ctx.getKey()} style={pdfStyles.horizontalRule} />
  ),

  hardBreak: (_node: TipTapNode, _children: ReactElement[], ctx: PDFRenderContext): ReactElement => (
    <Text key={ctx.getKey()}>{"\n"}</Text>
  ),

  image: (node, _children, ctx) => renderImage(node, ctx, false),
  imageComponent: (node, _children, ctx) => renderImage(node, ctx, true),

  calloutComponent: (node: TipTapNode, children: ReactElement[], ctx: PDFRenderContext): ReactElement => {
    const backgroundKey = stringAttribute(node.attrs?.["data-background"]) || "gray";
    const backgroundColor =
      (Object.hasOwn(backgroundColors, backgroundKey) ? backgroundColors[backgroundKey] : undefined) ||
      BACKGROUND_COLORS.layer3;

    return (
      <View key={ctx.getKey()} style={[pdfStyles.callout, { backgroundColor }]}>
        <View style={pdfStyles.calloutIconContainer}>{getCalloutIcon(node, TEXT_COLORS.primary)}</View>
        <View style={[pdfStyles.calloutContent, { color: TEXT_COLORS.primary }]}>{children}</View>
      </View>
    );
  },

  mention: (node: TipTapNode, _children: ReactElement[], ctx: PDFRenderContext): ReactElement => {
    const id = stringAttribute(node.attrs?.id) || "";
    const entityIdentifier = stringAttribute(node.attrs?.entity_identifier) || "";
    const entityName = stringAttribute(node.attrs?.entity_name) || "";

    let displayText = entityName || id || entityIdentifier;

    if (ctx.metadata && (entityName === "user_mention" || entityName === "user")) {
      const userMention = ctx.metadata.userMentions?.find((u) => u.id === entityIdentifier || u.id === id);
      if (userMention) {
        displayText = userMention.display_name;
      }
    }

    return (
      <Text key={ctx.getKey()} style={pdfStyles.mention}>
        @{displayText}
      </Text>
    );
  },
};

type InternalRenderContext = {
  parentType?: string;
  nestingLevel: number;
  listItemIndex: number;
  textAlign?: string | null;
  pdfContext: PDFRenderContext;
};

const renderNodeWithContext = (node: TipTapNode, context: InternalRenderContext): ReactElement => {
  const { parentType, nestingLevel, listItemIndex, textAlign, pdfContext } = context;

  const isListContainer = node.type === CORE_EXTENSIONS.BULLET_LIST || node.type === CORE_EXTENSIONS.ORDERED_LIST;

  let childTextAlign = textAlign;
  if (node.type === CORE_EXTENSIONS.PARAGRAPH && typeof node.attrs?.textAlign === "string") {
    childTextAlign = node.attrs.textAlign;
  }

  const renderContext: PDFRenderContext = {
    ...pdfContext,
    layout: {
      parentType,
      nestingLevel,
      listItemIndex,
      textAlign: childTextAlign,
      isHeader: node.content?.some((child) => child.type === CORE_EXTENSIONS.TABLE_HEADER) ?? false,
    },
  };

  let childNestingLevel = nestingLevel;
  if (isListContainer && parentType === CORE_EXTENSIONS.LIST_ITEM) {
    childNestingLevel = nestingLevel + 1;
  }

  let currentListItemIndex = 0;
  const children: ReactElement[] =
    node.content?.map((child) => {
      const childContext: InternalRenderContext = {
        parentType: node.type,
        nestingLevel: childNestingLevel,
        listItemIndex: 0,
        textAlign: childTextAlign,
        pdfContext,
      };

      if (isListContainer && child.type === CORE_EXTENSIONS.LIST_ITEM) {
        currentListItemIndex++;
        childContext.listItemIndex = currentListItemIndex;
      }

      return renderNodeWithContext(child, childContext);
    }) || [];

  const renderer = Object.hasOwn(nodeRenderers, node.type) ? nodeRenderers[node.type] : undefined;
  if (renderer) {
    return renderer(node, children, renderContext);
  }

  if (children.length > 0) {
    return <View key={pdfContext.getKey()}>{children}</View>;
  }

  return <View key={pdfContext.getKey()} />;
};

export const renderNode = (
  node: TipTapNode,
  parentType?: string,
  _index?: number,
  metadata?: PDFExportMetadata,
  getKey?: KeyGenerator
): ReactElement => {
  const keyGen = getKey ?? createKeyGenerator();

  return renderNodeWithContext(node, {
    parentType,
    nestingLevel: 0,
    listItemIndex: 0,
    pdfContext: { getKey: keyGen, metadata },
  });
};
