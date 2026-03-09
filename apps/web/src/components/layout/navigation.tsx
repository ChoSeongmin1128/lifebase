"use client";

import {
  Calendar,
  CheckCircle2,
  Cloud,
  House,
  Image as ImageIcon,
  Settings,
  type LucideIcon,
} from "lucide-react";
import { CLOUD_SECTION_ITEMS, parseCloudSection } from "@/lib/cloud-sections";

export interface AppNavItem {
  href: string;
  label: string;
  icon: LucideIcon;
  hasSubnav?: boolean;
}

export interface AppSubnavItem {
  href: string;
  label: string;
  isActive: boolean;
}

export const APP_NAV_ITEMS: AppNavItem[] = [
  { href: "/home", label: "Home", icon: House, hasSubnav: true },
  { href: "/cloud", label: "Cloud", icon: Cloud, hasSubnav: true },
  { href: "/calendar", label: "Calendar", icon: Calendar, hasSubnav: true },
  { href: "/todo", label: "Todo", icon: CheckCircle2, hasSubnav: true },
  { href: "/gallery", label: "Gallery", icon: ImageIcon, hasSubnav: true },
  { href: "/settings/general", label: "Settings", icon: Settings, hasSubnav: true },
] as const;

const HOME_FOCUS_ITEMS = [
  { key: "summary", label: "오늘 요약" },
  { key: "calendar", label: "일정" },
  { key: "todo", label: "Todo" },
  { key: "files", label: "파일" },
  { key: "storage", label: "저장공간" },
] as const;

const TODO_SCOPE_ITEMS = [
  { key: "all", label: "전체" },
  { key: "due", label: "기한 있음" },
  { key: "starred", label: "별표" },
  { key: "completed", label: "완료" },
] as const;

const GALLERY_MEDIA_ITEMS = [
  { key: "all", label: "전체" },
  { key: "image", label: "이미지" },
  { key: "video", label: "동영상" },
] as const;

const SETTINGS_SECTION_ITEMS = [
  { key: "general", label: "일반" },
  { key: "calendar", label: "캘린더" },
  { key: "todo", label: "Todo" },
  { key: "notifications", label: "알림" },
  { key: "cloud", label: "Cloud" },
] as const;

type CalendarViewMode = "month" | "week" | "3day" | "agenda" | "year-compact" | "year-timeline";

function normalizeSettingsSection(value: string | null | undefined) {
  if (!value) return "general";
  if (SETTINGS_SECTION_ITEMS.some((item) => item.key === value)) return value;
  if (value === "notification") return "notifications";
  return "general";
}

function buildSearchHref(pathname: string, searchParams: URLSearchParams) {
  const query = searchParams.toString();
  return query ? `${pathname}?${query}` : pathname;
}

function getCalendarDateFromPath(pathname: string): Date {
  const segments = pathname.split("/").filter(Boolean);
  const dateStr = segments[2];
  if (!dateStr) return new Date();

  if (/^\d{4}$/.test(dateStr)) {
    return new Date(Number.parseInt(dateStr, 10), 0, 1);
  }
  if (/^\d{4}-\d{2}$/.test(dateStr)) {
    const [year, month] = dateStr.split("-").map(Number);
    return new Date(year, month - 1, 1);
  }
  if (/^\d{4}-W\d{2}$/.test(dateStr)) {
    const [year, week] = dateStr.split("-W").map(Number);
    return getDateOfISOWeek(week, year);
  }
  if (/^\d{4}-\d{2}-\d{2}$/.test(dateStr)) {
    return new Date(`${dateStr}T00:00:00`);
  }
  return new Date();
}

function getDateOfISOWeek(week: number, year: number) {
  const jan4 = new Date(year, 0, 4);
  const dayOfWeek = jan4.getDay() || 7;
  const firstMonday = new Date(jan4);
  firstMonday.setDate(jan4.getDate() - dayOfWeek + 1);
  const result = new Date(firstMonday);
  result.setDate(firstMonday.getDate() + (week - 1) * 7);
  return result;
}

function buildCalendarHref(view: CalendarViewMode, date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");

  switch (view) {
    case "year-compact":
    case "year-timeline":
      return `/calendar/${view}/${year}`;
    case "month":
      return `/calendar/month/${year}-${month}`;
    case "week": {
      const jan4 = new Date(year, 0, 4);
      const dayOfYear = Math.floor((date.getTime() - new Date(year, 0, 1).getTime()) / 86400000) + 1;
      const weekNum = Math.ceil((dayOfYear + jan4.getDay()) / 7);
      return `/calendar/week/${year}-W${String(weekNum).padStart(2, "0")}`;
    }
    case "3day":
      return `/calendar/3day/${year}-${month}-${day}`;
    case "agenda":
      return "/calendar/agenda";
    default:
      return "/calendar";
  }
}

function buildHomeSubnav(searchParams: URLSearchParams): AppSubnavItem[] {
  const focus = searchParams.get("focus") || "summary";
  return HOME_FOCUS_ITEMS.map((item) => {
    const params = new URLSearchParams(searchParams.toString());
    if (item.key === "summary") {
      params.delete("focus");
    } else {
      params.set("focus", item.key);
    }
    return {
      href: buildSearchHref("/home", params),
      label: item.label,
      isActive: focus === item.key || (!searchParams.get("focus") && item.key === "summary"),
    };
  });
}

function buildCloudSubnav(searchParams: URLSearchParams): AppSubnavItem[] {
  const currentSection = parseCloudSection(searchParams.get("section"));
  return CLOUD_SECTION_ITEMS.map((item) => {
    const params = new URLSearchParams(searchParams.toString());
    if (item.section) {
      params.set("section", item.section);
    } else {
      params.delete("section");
    }
    return {
      href: buildSearchHref("/cloud", params),
      label: item.label,
      isActive: currentSection === item.section,
    };
  });
}

function buildCalendarSubnav(pathname: string): AppSubnavItem[] {
  const currentView = (pathname.split("/").filter(Boolean)[1] || "month") as CalendarViewMode;
  const date = getCalendarDateFromPath(pathname);
  return [
    { key: "month", label: "월간" },
    { key: "week", label: "주간" },
    { key: "3day", label: "3일" },
    { key: "agenda", label: "일정" },
    { key: "year-compact", label: "연간" },
    { key: "year-timeline", label: "타임라인" },
  ].map((item) => ({
    href: buildCalendarHref(item.key as CalendarViewMode, date),
    label: item.label,
    isActive: currentView === item.key,
  }));
}

function buildTodoSubnav(searchParams: URLSearchParams): AppSubnavItem[] {
  const scope = searchParams.get("scope") || "all";
  return TODO_SCOPE_ITEMS.map((item) => {
    const params = new URLSearchParams(searchParams.toString());
    if (item.key === "all") {
      params.delete("scope");
    } else {
      params.set("scope", item.key);
    }
    return {
      href: buildSearchHref("/todo", params),
      label: item.label,
      isActive: scope === item.key || (!searchParams.get("scope") && item.key === "all"),
    };
  });
}

function buildGallerySubnav(searchParams: URLSearchParams): AppSubnavItem[] {
  const media = searchParams.get("media") || "all";
  return GALLERY_MEDIA_ITEMS.map((item) => {
    const params = new URLSearchParams(searchParams.toString());
    if (item.key === "all") {
      params.delete("media");
    } else {
      params.set("media", item.key);
    }
    return {
      href: buildSearchHref("/gallery", params),
      label: item.label,
      isActive: media === item.key || (!searchParams.get("media") && item.key === "all"),
    };
  });
}

function buildSettingsSubnav(pathname: string): AppSubnavItem[] {
  const section = normalizeSettingsSection(pathname.split("/").filter(Boolean)[1]);
  return SETTINGS_SECTION_ITEMS.map((item) => ({
    href: `/settings/${item.key}`,
    label: item.label,
    isActive: section === item.key,
  }));
}

export function isNavItemActive(pathname: string, href: string) {
  const normalizedHref = href === "/settings/general" ? "/settings" : href;
  return pathname === normalizedHref || pathname.startsWith(`${normalizedHref}/`);
}

export function getSidebarSubnavItems(pathname: string, searchParams: URLSearchParams): AppSubnavItem[] {
  if (pathname.startsWith("/home")) return buildHomeSubnav(searchParams);
  if (pathname.startsWith("/cloud")) return buildCloudSubnav(searchParams);
  if (pathname.startsWith("/calendar")) return buildCalendarSubnav(pathname);
  if (pathname.startsWith("/todo")) return buildTodoSubnav(searchParams);
  if (pathname.startsWith("/gallery")) return buildGallerySubnav(searchParams);
  if (pathname.startsWith("/settings")) return buildSettingsSubnav(pathname);
  return [];
}

export function normalizeSettingsHref(href: string) {
  return href === "/settings" ? "/settings/general" : href;
}
