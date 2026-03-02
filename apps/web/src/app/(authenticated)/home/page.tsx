"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { PageToolbar, PageToolbarGroup } from "@/components/layout/PageToolbar";
import { useHomeActions } from "@/features/home/ui/hooks/useHomeActions";
import type { HomeSummary, HomeSummaryEvent, HomeSummaryTodo } from "@/features/home/domain/HomeSummary";
import { CalendarPlus, CheckCircle2, Upload } from "lucide-react";

const STORAGE_TYPE_META = {
  image: { label: "이미지", color: "#22c55e" },
  video: { label: "비디오", color: "#3b82f6" },
  other: { label: "기타", color: "#f59e0b" },
} as const;

function toLocalRFC3339(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  const seconds = String(date.getSeconds()).padStart(2, "0");

  const offsetMinutes = -date.getTimezoneOffset();
  const sign = offsetMinutes >= 0 ? "+" : "-";
  const abs = Math.abs(offsetMinutes);
  const offsetHour = String(Math.floor(abs / 60)).padStart(2, "0");
  const offsetMin = String(abs % 60).padStart(2, "0");

  return `${year}-${month}-${day}T${hours}:${minutes}:${seconds}${sign}${offsetHour}:${offsetMin}`;
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

function formatEventTime(event: HomeSummaryEvent): string {
  const start = new Date(event.start_time);
  if (event.is_all_day) {
    return "종일";
  }
  return start.toLocaleTimeString("ko-KR", { hour: "numeric", minute: "2-digit", hour12: true });
}

function getTodoBadge(todo: HomeSummaryTodo): string {
  if (todo.priority === "urgent") return "긴급";
  if (todo.priority === "high") return "높음";
  return todo.due_date ? todo.due_date : "보통";
}

export default function HomePage() {
  const { getSummary } = useHomeActions();
  const [summary, setSummary] = useState<HomeSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const queryRange = useMemo(() => {
    const now = new Date();
    const start = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 0, 0, 0, 0);
    const end = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1, 0, 0, 0, 0);
    return {
      start: toLocalRFC3339(start),
      end: toLocalRFC3339(end),
    };
  }, []);

  const loadSummary = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const next = await getSummary({
        start: queryRange.start,
        end: queryRange.end,
      });
      setSummary(next);
    } catch {
      setError("Home 요약을 불러오지 못했습니다.");
      setSummary(null);
    } finally {
      setLoading(false);
    }
  }, [getSummary, queryRange.end, queryRange.start]);

  useEffect(() => {
    loadSummary();
  }, [loadSummary]);

  const usedBytes = summary?.storage.used_bytes ?? 0;
  const quotaBytes = summary?.storage.quota_bytes ?? 0;
  const usagePercent = Math.min(100, Math.max(0, summary?.storage.usage_percent ?? 0));
  const todayEventCount = summary?.events.total_count ?? 0;
  const overdueTodoCount = summary?.todos.overdue_count ?? 0;
  const now = new Date();
  const pastEvents = (summary?.events.items || []).filter((event) => new Date(event.end_time) < now);
  const todayEvents = (summary?.events.items || []).filter((event) => new Date(event.end_time) >= now);
  const storageBreakdown = (["image", "video", "other"] as const).map((type) => {
    const found = summary?.storage.breakdown.find((item) => item.type === type);
    return {
      type,
      label: STORAGE_TYPE_META[type].label,
      color: STORAGE_TYPE_META[type].color,
      bytes: found?.bytes || 0,
      percent: found?.percent || 0,
    };
  });
  const storageConicGradient = (() => {
    if (usedBytes <= 0) {
      return "conic-gradient(#e5e7eb 0% 100%)";
    }
    const segments: string[] = [];
    let offset = 0;
    for (const item of storageBreakdown) {
      const pct = Math.max(0, item.percent);
      if (pct <= 0) continue;
      const end = Math.min(100, offset + pct);
      segments.push(`${item.color} ${offset}% ${end}%`);
      offset = end;
    }
    if (offset < 100) {
      segments.push(`#e5e7eb ${offset}% 100%`);
    }
    return `conic-gradient(${segments.join(", ")})`;
  })();

  return (
    <div className="flex h-full flex-col">
      <PageToolbar>
        <PageToolbarGroup>
          <h2 className="text-lg font-semibold text-text-strong">Home</h2>
          <span className="text-sm text-text-muted">오늘 요약</span>
        </PageToolbarGroup>
        <Button variant="secondary" size="sm" onClick={loadSummary}>
          새로고침
        </Button>
      </PageToolbar>

      <div className="flex-1 overflow-auto p-4 md:p-6">
        {loading ? (
          <div className="mx-auto grid w-full max-w-[1200px] gap-4 md:grid-cols-2 xl:grid-cols-3">
            {Array.from({ length: 5 }).map((_, idx) => (
              <div key={idx} className="h-40 animate-pulse rounded-xl border border-border bg-surface-accent/40" />
            ))}
          </div>
        ) : error ? (
          <div className="mx-auto flex w-full max-w-[1200px] flex-col items-center justify-center gap-3 rounded-xl border border-border bg-surface px-6 py-12 text-center">
            <p className="text-sm text-error">{error}</p>
            <Button variant="secondary" size="sm" onClick={loadSummary}>다시 시도</Button>
          </div>
        ) : (
          <div className="mx-auto w-full max-w-[1200px] space-y-4">
            <section className="rounded-xl border border-border bg-surface p-4">
              <div className="mb-3 flex items-center justify-between">
                <h3 className="text-sm font-semibold text-text-strong">오늘 한눈에</h3>
              </div>
              <div className="grid gap-2 sm:grid-cols-3">
                <div className="rounded-lg bg-surface-accent/60 px-3 py-2">
                  <p className="text-xs text-text-muted">오늘 일정</p>
                  <p className="tabular-nums text-lg font-semibold text-text-strong">{todayEventCount}</p>
                </div>
                <div className="rounded-lg bg-surface-accent/60 px-3 py-2">
                  <p className="text-xs text-text-muted">지연 Todo</p>
                  <p className="tabular-nums text-lg font-semibold text-text-strong">{overdueTodoCount}</p>
                </div>
                <div className="rounded-lg bg-surface-accent/60 px-3 py-2">
                  <p className="text-xs text-text-muted">저장공간</p>
                  <p className="tabular-nums text-lg font-semibold text-text-strong">{usagePercent.toFixed(1)}%</p>
                </div>
              </div>
            </section>

            <section className="rounded-xl border border-border bg-surface p-4">
              <div className="mb-3 flex items-center justify-between">
                <h3 className="text-sm font-semibold text-text-strong">빠른 액션</h3>
              </div>
              <div className="grid gap-2 sm:grid-cols-3">
                <Link
                  href="/calendar?quick=create"
                  className="inline-flex h-9 items-center justify-start gap-2 rounded-lg border border-border bg-background px-4 text-sm font-medium text-text-primary transition-colors hover:bg-surface-accent"
                >
                  <CalendarPlus size={16} />
                  일정 추가
                </Link>
                <Link
                  href="/todo?quick=create"
                  className="inline-flex h-9 items-center justify-start gap-2 rounded-lg border border-border bg-background px-4 text-sm font-medium text-text-primary transition-colors hover:bg-surface-accent"
                >
                  <CheckCircle2 size={16} />
                  Todo 추가
                </Link>
                <Link
                  href="/cloud?quick=upload"
                  className="inline-flex h-9 items-center justify-start gap-2 rounded-lg border border-border bg-background px-4 text-sm font-medium text-text-primary transition-colors hover:bg-surface-accent"
                >
                  <Upload size={16} />
                  파일 업로드
                </Link>
              </div>
            </section>

            <div className="grid gap-4 xl:grid-cols-12 xl:items-stretch">
              <div className="space-y-4 xl:col-span-8 xl:grid xl:grid-rows-2 xl:gap-4 xl:space-y-0">
                <section className="rounded-xl border border-border bg-surface p-4 xl:h-full min-h-[220px] flex flex-col">
                  <div className="mb-3 flex items-center justify-between">
                    <h3 className="text-sm font-semibold text-text-strong">캘린더</h3>
                    <Link href="/calendar" className="text-xs text-primary">전체 보기</Link>
                  </div>
                  <div className="grid gap-3 md:grid-cols-2 xl:flex-1 xl:overflow-auto xl:pr-1">
                    <div>
                      <p className="mb-1 text-xs font-medium text-text-muted">지난 일정</p>
                      <div className="space-y-2">
                        {pastEvents.length ? pastEvents.slice(0, 4).map((event) => (
                          <div key={event.id} className="flex items-start gap-2 rounded-lg bg-background px-2 py-1.5">
                            <span
                              className="mt-1 h-2 w-2 shrink-0 rounded-full"
                              style={{ backgroundColor: event.color_id || "#4285F4" }}
                            />
                            <div className="min-w-0">
                              <p className="truncate text-sm text-text-primary">{event.title || "(제목 없음)"}</p>
                              <p className="text-xs text-text-muted">{formatEventTime(event)}</p>
                            </div>
                          </div>
                        )) : (
                          <p className="text-sm text-text-muted">지난 일정이 없습니다.</p>
                        )}
                      </div>
                    </div>

                    <div>
                      <p className="mb-1 text-xs font-medium text-text-muted">오늘 일정</p>
                      <div className="space-y-2">
                        {todayEvents.length ? todayEvents.slice(0, 4).map((event) => (
                          <div key={event.id} className="flex items-start gap-2 rounded-lg bg-background px-2 py-1.5">
                            <span
                              className="mt-1 h-2 w-2 shrink-0 rounded-full"
                              style={{ backgroundColor: event.color_id || "#4285F4" }}
                            />
                            <div className="min-w-0">
                              <p className="truncate text-sm text-text-primary">{event.title || "(제목 없음)"}</p>
                              <p className="text-xs text-text-muted">{formatEventTime(event)}</p>
                            </div>
                          </div>
                        )) : (
                          <p className="text-sm text-text-muted">오늘 일정이 없습니다.</p>
                        )}
                      </div>
                    </div>
                  </div>
                </section>

                <section className="rounded-xl border border-border bg-surface p-4 xl:h-full min-h-[220px] flex flex-col">
                  <div className="mb-3 flex items-center justify-between">
                    <h3 className="text-sm font-semibold text-text-strong">Todo</h3>
                    <Link href="/todo" className="text-xs text-primary">전체 보기</Link>
                  </div>
                  <div className="grid gap-3 md:grid-cols-2 xl:flex-1 xl:overflow-auto xl:pr-1">
                    <div>
                      <p className="mb-1 text-xs font-medium text-text-muted">지난 Todo</p>
                      <div className="space-y-2">
                        {summary?.todos.overdue.length ? summary.todos.overdue.slice(0, 4).map((todo) => (
                          <div key={todo.id} className="flex items-center justify-between gap-2 rounded-lg bg-background px-2 py-1.5">
                            <p className="truncate text-sm text-text-primary">{todo.title}</p>
                            <span className="shrink-0 text-xs text-text-muted">{getTodoBadge(todo)}</span>
                          </div>
                        )) : (
                          <p className="text-sm text-text-muted">지난 Todo가 없습니다.</p>
                        )}
                      </div>
                    </div>

                    <div>
                      <p className="mb-1 text-xs font-medium text-text-muted">오늘 Todo</p>
                      <div className="space-y-2">
                        {summary?.todos.today.length ? summary.todos.today.slice(0, 4).map((todo) => (
                          <div key={todo.id} className="flex items-center justify-between gap-2 rounded-lg bg-background px-2 py-1.5">
                            <p className="truncate text-sm text-text-primary">{todo.title}</p>
                            <span className="shrink-0 text-xs text-text-muted">{getTodoBadge(todo)}</span>
                          </div>
                        )) : (
                          <p className="text-sm text-text-muted">오늘 Todo가 없습니다.</p>
                        )}
                      </div>
                    </div>
                  </div>
                </section>
              </div>

              <div className="space-y-4 xl:col-span-4 xl:grid xl:grid-rows-2 xl:gap-4 xl:space-y-0">
                <section className="rounded-xl border border-border bg-surface p-4 xl:h-full min-h-[220px] flex flex-col">
                  <div className="mb-3 flex items-center justify-between">
                    <h3 className="text-sm font-semibold text-text-strong">최근 파일</h3>
                    <Link href="/cloud?section=recent" className="text-xs text-primary">전체 보기</Link>
                  </div>
                  <div className="space-y-2 xl:flex-1 xl:overflow-auto xl:pr-1">
                    {summary?.files.recent.length ? summary.files.recent.map((file) => (
                      <div key={file.id} className="flex items-center justify-between gap-2 rounded-lg bg-background px-2 py-1.5">
                        <p className="truncate text-sm text-text-primary">{file.name}</p>
                        <span className="shrink-0 text-xs text-text-muted">{formatBytes(file.size_bytes)}</span>
                      </div>
                    )) : (
                      <p className="text-sm text-text-muted">최근 파일이 없습니다.</p>
                    )}
                  </div>
                </section>

                <section className="rounded-xl border border-border bg-surface p-4 xl:h-full min-h-[220px] flex flex-col">
                  <div className="mb-3 flex items-center justify-between">
                    <h3 className="text-sm font-semibold text-text-strong">저장공간</h3>
                    <Link href="/cloud" className="text-xs text-primary">Cloud 열기</Link>
                  </div>
                  <p className="mb-3 text-sm text-text-secondary">
                    <span className="tabular-nums">{formatBytes(usedBytes)}</span>
                    <span className="mx-1 text-text-muted">/</span>
                    <span className="tabular-nums">{formatBytes(quotaBytes)}</span>
                  </p>
                  <div className="flex items-center gap-4">
                    <div
                      className="relative h-24 w-24 shrink-0 rounded-full"
                      style={{ background: storageConicGradient }}
                    >
                      <div className="absolute inset-[10px] flex items-center justify-center rounded-full bg-surface">
                        <span className="text-xs font-medium text-text-strong tabular-nums">
                          {usagePercent.toFixed(1)}%
                        </span>
                      </div>
                    </div>
                    <div className="flex-1 space-y-1.5">
                      {storageBreakdown.map((item) => (
                        <div key={item.type} className="flex items-center justify-between gap-2 text-xs">
                          <div className="flex items-center gap-2 min-w-0">
                            <span
                              className="h-2 w-2 shrink-0 rounded-full"
                              style={{ backgroundColor: item.color }}
                            />
                            <span className="text-text-secondary">{item.label}</span>
                          </div>
                          <span className="text-text-muted tabular-nums">
                            {item.percent.toFixed(1)}% · {formatBytes(item.bytes)}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                </section>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
