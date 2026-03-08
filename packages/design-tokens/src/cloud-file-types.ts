export type CloudItemType = "folder" | "file";

export type CloudFileTypeKey =
  | "folder"
  | "image"
  | "video"
  | "audio"
  | "pdf"
  | "spreadsheet"
  | "presentation"
  | "code"
  | "document"
  | "archive"
  | "unknown";

export interface CloudFileTypeToken {
  key: CloudFileTypeKey;
  label: string;
  foreground: string;
  background: string;
}

const CLOUD_FILE_TYPE_TOKENS: Record<CloudFileTypeKey, CloudFileTypeToken> = {
  folder: {
    key: "folder",
    label: "FD",
    foreground: "#d97706",
    background: "rgba(217,119,6,0.12)",
  },
  image: {
    key: "image",
    label: "IMG",
    foreground: "#16a34a",
    background: "rgba(22,163,74,0.12)",
  },
  video: {
    key: "video",
    label: "VID",
    foreground: "#e11d48",
    background: "rgba(225,29,72,0.12)",
  },
  audio: {
    key: "audio",
    label: "AUD",
    foreground: "#7c3aed",
    background: "rgba(124,58,237,0.12)",
  },
  pdf: {
    key: "pdf",
    label: "PDF",
    foreground: "#dc2626",
    background: "rgba(220,38,38,0.12)",
  },
  spreadsheet: {
    key: "spreadsheet",
    label: "XLS",
    foreground: "#15803d",
    background: "rgba(21,128,61,0.12)",
  },
  presentation: {
    key: "presentation",
    label: "PPT",
    foreground: "#ea580c",
    background: "rgba(234,88,12,0.12)",
  },
  code: {
    key: "code",
    label: "</>",
    foreground: "#0284c7",
    background: "rgba(2,132,199,0.12)",
  },
  document: {
    key: "document",
    label: "DOC",
    foreground: "#2563eb",
    background: "rgba(37,99,235,0.12)",
  },
  archive: {
    key: "archive",
    label: "ZIP",
    foreground: "#64748b",
    background: "rgba(100,116,139,0.12)",
  },
  unknown: {
    key: "unknown",
    label: "FILE",
    foreground: "#64748b",
    background: "rgba(100,116,139,0.12)",
  },
};

const MIME_RULES: Array<{ pattern: RegExp; key: CloudFileTypeKey }> = [
  { pattern: /^image\//, key: "image" },
  { pattern: /^video\//, key: "video" },
  { pattern: /^audio\//, key: "audio" },
  { pattern: /pdf/, key: "pdf" },
  { pattern: /spreadsheet|excel|csv/, key: "spreadsheet" },
  { pattern: /presentation|powerpoint/, key: "presentation" },
  { pattern: /zip|archive|compressed|tar|rar|7z/, key: "archive" },
  { pattern: /javascript|typescript|json|html|css|xml|python|java|go|rust/, key: "code" },
  { pattern: /^text\/|document|msword|wordprocessingml|rtf/, key: "document" },
];

export function getCloudFileTypeKey(mimeType?: string | null): CloudFileTypeKey {
  const normalizedMimeType = mimeType?.toLowerCase() ?? "";

  for (const rule of MIME_RULES) {
    if (rule.pattern.test(normalizedMimeType)) {
      return rule.key;
    }
  }

  return "unknown";
}

export function getCloudItemToken(input: {
  type: CloudItemType;
  mimeType?: string | null;
}): CloudFileTypeToken {
  if (input.type === "folder") {
    return CLOUD_FILE_TYPE_TOKENS.folder;
  }

  return CLOUD_FILE_TYPE_TOKENS[getCloudFileTypeKey(input.mimeType)];
}

export { CLOUD_FILE_TYPE_TOKENS };
