export const CLOUD_SECTIONS = ["", "recent", "shared", "starred", "trash"] as const;

export type CloudSection = (typeof CLOUD_SECTIONS)[number];

export const CLOUD_SECTION_LABELS: Record<CloudSection, string> = {
  "": "내 파일",
  recent: "최근",
  shared: "공유됨",
  starred: "중요",
  trash: "휴지통",
};

export const CLOUD_SECTION_ITEMS = CLOUD_SECTIONS.map((section) => ({
  section,
  label: CLOUD_SECTION_LABELS[section],
}));

export function parseCloudSection(value: string | null): CloudSection {
  if (value && (CLOUD_SECTIONS as readonly string[]).includes(value)) {
    return value as CloudSection;
  }
  return "";
}
