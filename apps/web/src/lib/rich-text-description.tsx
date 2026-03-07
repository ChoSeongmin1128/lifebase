"use client";

import { useMemo } from "react";

import { cn } from "@/lib/utils";

const ALLOWED_TAGS = new Set([
  "A",
  "B",
  "BLOCKQUOTE",
  "BR",
  "CODE",
  "EM",
  "H1",
  "H2",
  "H3",
  "H4",
  "H5",
  "H6",
  "I",
  "LI",
  "OL",
  "P",
  "PRE",
  "STRONG",
  "U",
  "UL",
]);

const BLOCKED_TAGS = new Set(["IFRAME", "LINK", "META", "OBJECT", "SCRIPT", "STYLE"]);

function decodeHtmlEntities(input: string): string {
  if (typeof document !== "undefined") {
    const textarea = document.createElement("textarea");
    textarea.innerHTML = input;
    return textarea.value;
  }

  return input
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&nbsp;/g, " ")
    .replace(/&amp;/g, "&");
}

function normalizeRichTextToPlainText(input: string): string {
  const decoded = decodeHtmlEntities(input);
  return decoded
    .replace(/<\s*br\s*\/?>/gi, "\n")
    .replace(/<\s*li[^>]*>/gi, "• ")
    .replace(/<\/\s*(p|div|h1|h2|h3|h4|h5|h6|li|ul|ol|pre|blockquote)\s*>/gi, "\n")
    .replace(/<[^>]+>/g, "")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}

function looksLikeRichText(input: string): boolean {
  const decoded = decodeHtmlEntities(input);
  return /<\s*(p|br|strong|em|b|i|u|ul|ol|li|h[1-6]|blockquote|pre|code|a)\b/i.test(decoded);
}

function sanitizeRichHtml(input: string): string | null {
  if (typeof DOMParser === "undefined") {
    return null;
  }

  const decoded = decodeHtmlEntities(input);
  const parser = new DOMParser();
  const doc = parser.parseFromString(decoded, "text/html");

  const walk = (node: Node) => {
    if (node.nodeType !== Node.ELEMENT_NODE) return;

    const element = node as HTMLElement;
    const tagName = element.tagName.toUpperCase();

    if (BLOCKED_TAGS.has(tagName)) {
      element.remove();
      return;
    }

    Array.from(element.childNodes).forEach(walk);

    if (!ALLOWED_TAGS.has(tagName)) {
      const fragment = doc.createDocumentFragment();
      while (element.firstChild) {
        fragment.appendChild(element.firstChild);
      }
      element.replaceWith(fragment);
      return;
    }

    Array.from(element.attributes).forEach((attribute) => {
      const name = attribute.name.toLowerCase();
      if (tagName === "A" && name === "href") {
        const href = attribute.value.trim();
        if (!/^(https?:|mailto:)/i.test(href)) {
          element.removeAttribute(attribute.name);
        }
        return;
      }
      if (tagName === "A" && (name === "target" || name === "rel")) {
        return;
      }
      element.removeAttribute(attribute.name);
    });

    if (tagName === "A" && element.getAttribute("href")) {
      element.setAttribute("target", "_blank");
      element.setAttribute("rel", "noopener noreferrer");
    }
  };

  Array.from(doc.body.childNodes).forEach(walk);
  return doc.body.innerHTML.trim();
}

export function RichTextDescription({
  value,
  className,
}: {
  value?: string | null;
  className?: string;
}) {
  const prepared = useMemo(() => {
    const raw = value?.trim();
    if (!raw) return null;

    if (!looksLikeRichText(raw)) {
      return { kind: "text" as const, text: normalizeRichTextToPlainText(raw) || raw };
    }

    const sanitized = sanitizeRichHtml(raw);
    if (!sanitized) {
      return { kind: "text" as const, text: normalizeRichTextToPlainText(raw) || raw };
    }

    return { kind: "html" as const, html: sanitized };
  }, [value]);

  if (!prepared) return null;

  const baseClassName = cn(
    "mt-2 text-xs leading-5 text-text-secondary whitespace-pre-wrap",
    "[&_a]:text-primary [&_a]:underline",
    "[&_blockquote]:border-l-2 [&_blockquote]:border-border [&_blockquote]:pl-3 [&_blockquote]:italic",
    "[&_code]:rounded [&_code]:bg-muted [&_code]:px-1 [&_code]:py-0.5",
    "[&_h1]:mt-3 [&_h1]:text-sm [&_h1]:font-semibold [&_h1]:text-text-strong",
    "[&_h2]:mt-3 [&_h2]:text-sm [&_h2]:font-semibold [&_h2]:text-text-strong",
    "[&_h3]:mt-3 [&_h3]:text-sm [&_h3]:font-semibold [&_h3]:text-text-strong",
    "[&_h4]:mt-3 [&_h4]:text-sm [&_h4]:font-semibold [&_h4]:text-text-strong",
    "[&_li]:ml-4 [&_li]:list-item",
    "[&_ol]:my-2 [&_ol]:list-decimal [&_ol]:pl-4",
    "[&_p]:mt-2 [&_p:first-child]:mt-0",
    "[&_pre]:overflow-x-auto [&_pre]:rounded [&_pre]:bg-muted [&_pre]:p-2",
    "[&_strong]:font-semibold [&_strong]:text-text-strong",
    "[&_ul]:my-2 [&_ul]:list-disc [&_ul]:pl-4",
    className,
  );

  if (prepared.kind === "html") {
    return <div className={baseClassName} dangerouslySetInnerHTML={{ __html: prepared.html }} />;
  }

  return <p className={baseClassName}>{prepared.text}</p>;
}

export function formatDescriptionText(value?: string | null): string {
  if (!value) return "";
  return normalizeRichTextToPlainText(value);
}
