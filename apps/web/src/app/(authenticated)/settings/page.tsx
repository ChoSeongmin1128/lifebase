"use client";

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";

type SettingsTab = "general" | "calendar" | "todo" | "notification" | "cloud";

const TABS: { value: SettingsTab; label: string }[] = [
  { value: "general", label: "일반" },
  { value: "calendar", label: "캘린더" },
  { value: "todo", label: "Todo" },
  { value: "notification", label: "알림" },
  { value: "cloud", label: "Cloud" },
];

export default function SettingsPage() {
  const [tab, setTab] = useState<SettingsTab>("general");
  const [settings, setSettings] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(true);

  const token = getAccessToken();

  const loadSettings = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    try {
      const data = await api<{ settings: Record<string, string> }>("/settings", { token });
      setSettings(data.settings || {});
    } catch {
      setSettings({});
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    loadSettings();
  }, [loadSettings]);

  const updateSetting = async (key: string, value: string) => {
    if (!token) return;
    setSettings((prev) => ({ ...prev, [key]: value }));
    try {
      await api("/settings", { method: "PATCH", body: { [key]: value }, token });
    } catch (err) {
      console.error("Update setting failed:", err);
    }
  };

  const get = (key: string, fallback: string = "") => settings[key] ?? fallback;

  return (
    <div className="flex h-full flex-col">
      {/* Tab bar */}
      <div className="flex overflow-x-auto border-b border-foreground/10 px-4 md:px-6">
        {TABS.map((t) => (
          <button
            key={t.value}
            onClick={() => setTab(t.value)}
            className={`shrink-0 px-3 md:px-4 py-3 text-sm border-b-2 transition-colors ${
              tab === t.value
                ? "border-foreground font-medium"
                : "border-transparent text-foreground/50 hover:text-foreground/70"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-4 md:p-6">
        {loading ? (
          <div className="flex items-center justify-center py-20 text-foreground/40">
            불러오는 중...
          </div>
        ) : (
          <div className="mx-auto max-w-2xl space-y-6">
            {tab === "general" && (
              <>
                <SettingsCard title="테마">
                  <SettingRow label="모드">
                    <SegmentedControl
                      value={get("theme", "system")}
                      options={[
                        { value: "light", label: "라이트" },
                        { value: "dark", label: "다크" },
                        { value: "system", label: "시스템" },
                      ]}
                      onChange={(v) => updateSetting("theme", v)}
                    />
                  </SettingRow>
                </SettingsCard>

                <SettingsCard title="계정">
                  <SettingRow label="Google 계정">
                    <span className="text-sm text-foreground/60">연결됨</span>
                  </SettingRow>
                </SettingsCard>
              </>
            )}

            {tab === "calendar" && (
              <>
                <SettingsCard title="캘린더 표시">
                  <SettingRow label="기본 뷰">
                    <select
                      value={get("calendar_default_view", "month")}
                      onChange={(e) => updateSetting("calendar_default_view", e.target.value)}
                      className="rounded border border-foreground/10 bg-background px-2 py-1 text-sm outline-none"
                    >
                      <option value="month">월간</option>
                      <option value="week">주간</option>
                      <option value="3day">3일</option>
                      <option value="agenda">일정</option>
                    </select>
                  </SettingRow>
                  <SettingRow label="주간 뷰 시작 시간">
                    <select
                      value={get("week_start_hour", "8")}
                      onChange={(e) => updateSetting("week_start_hour", e.target.value)}
                      className="rounded border border-foreground/10 bg-background px-2 py-1 text-sm outline-none"
                    >
                      {Array.from({ length: 24 }, (_, i) => (
                        <option key={i} value={String(i)}>
                          {String(i).padStart(2, "0")}:00
                        </option>
                      ))}
                    </select>
                  </SettingRow>
                  <SettingRow label="주간 뷰 종료 시간">
                    <select
                      value={get("week_end_hour", "22")}
                      onChange={(e) => updateSetting("week_end_hour", e.target.value)}
                      className="rounded border border-foreground/10 bg-background px-2 py-1 text-sm outline-none"
                    >
                      {Array.from({ length: 25 }, (_, i) => (
                        <option key={i} value={String(i)}>
                          {String(i).padStart(2, "0")}:00
                        </option>
                      ))}
                    </select>
                  </SettingRow>
                </SettingsCard>
              </>
            )}

            {tab === "todo" && (
              <>
                <SettingsCard title="Todo 설정">
                  <SettingRow label="기본 정렬">
                    <select
                      value={get("todo_default_sort", "due")}
                      onChange={(e) => updateSetting("todo_default_sort", e.target.value)}
                      className="rounded border border-foreground/10 bg-background px-2 py-1 text-sm outline-none"
                    >
                      <option value="due">마감일순</option>
                      <option value="priority">우선순위순</option>
                      <option value="created_at">생성일순</option>
                      <option value="manual">수동 정렬</option>
                    </select>
                  </SettingRow>
                  <SettingRow label="캘린더에 완료된 Todo 표시">
                    <SegmentedControl
                      value={get("show_done_in_calendar", "false")}
                      options={[
                        { value: "true", label: "표시" },
                        { value: "false", label: "숨김" },
                      ]}
                      onChange={(v) => updateSetting("show_done_in_calendar", v)}
                    />
                  </SettingRow>
                </SettingsCard>
              </>
            )}

            {tab === "notification" && (
              <>
                <SettingsCard title="알림">
                  <SettingRow label="Push 알림">
                    <SegmentedControl
                      value={get("push_enabled", "true")}
                      options={[
                        { value: "true", label: "켜짐" },
                        { value: "false", label: "꺼짐" },
                      ]}
                      onChange={(v) => updateSetting("push_enabled", v)}
                    />
                  </SettingRow>
                  <SettingRow label="방해 금지 시작">
                    <select
                      value={get("dnd_start", "23")}
                      onChange={(e) => updateSetting("dnd_start", e.target.value)}
                      className="rounded border border-foreground/10 bg-background px-2 py-1 text-sm outline-none"
                    >
                      {Array.from({ length: 24 }, (_, i) => (
                        <option key={i} value={String(i)}>
                          {String(i).padStart(2, "0")}:00
                        </option>
                      ))}
                    </select>
                  </SettingRow>
                  <SettingRow label="방해 금지 종료">
                    <select
                      value={get("dnd_end", "7")}
                      onChange={(e) => updateSetting("dnd_end", e.target.value)}
                      className="rounded border border-foreground/10 bg-background px-2 py-1 text-sm outline-none"
                    >
                      {Array.from({ length: 25 }, (_, i) => (
                        <option key={i} value={String(i)}>
                          {String(i).padStart(2, "0")}:00
                        </option>
                      ))}
                    </select>
                  </SettingRow>
                  <SettingRow label="긴급만 허용">
                    <SegmentedControl
                      value={get("dnd_urgent_only", "false")}
                      options={[
                        { value: "true", label: "켜짐" },
                        { value: "false", label: "꺼짐" },
                      ]}
                      onChange={(v) => updateSetting("dnd_urgent_only", v)}
                    />
                  </SettingRow>
                </SettingsCard>
              </>
            )}

            {tab === "cloud" && (
              <>
                <SettingsCard title="스토리지">
                  <SettingRow label="사용량">
                    <span className="text-sm text-foreground/60">계산 중...</span>
                  </SettingRow>
                  <SettingRow label="할당량">
                    <span className="text-sm text-foreground/60">1 TB</span>
                  </SettingRow>
                </SettingsCard>
                <SettingsCard title="Cloud 정렬">
                  <SettingRow label="기본 정렬">
                    <select
                      value={get("cloud_default_sort", "name")}
                      onChange={(e) => updateSetting("cloud_default_sort", e.target.value)}
                      className="rounded border border-foreground/10 bg-background px-2 py-1 text-sm outline-none"
                    >
                      <option value="name">이름순</option>
                      <option value="updated_at">수정일순</option>
                      <option value="created_at">생성일순</option>
                      <option value="size">크기순</option>
                    </select>
                  </SettingRow>
                </SettingsCard>
              </>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function SettingsCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-foreground/10 p-4">
      <h3 className="mb-3 text-sm font-medium">{title}</h3>
      <div className="space-y-3">{children}</div>
    </div>
  );
}

function SettingRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-sm text-foreground/70">{label}</span>
      {children}
    </div>
  );
}

function SegmentedControl({
  value,
  options,
  onChange,
}: {
  value: string;
  options: { value: string; label: string }[];
  onChange: (value: string) => void;
}) {
  return (
    <div className="flex rounded-md border border-foreground/10">
      {options.map((opt) => (
        <button
          key={opt.value}
          onClick={() => onChange(opt.value)}
          className={`px-3 py-1 text-xs ${
            value === opt.value
              ? "bg-foreground/10 font-medium"
              : "hover:bg-foreground/5 text-foreground/60"
          }`}
        >
          {opt.label}
        </button>
      ))}
    </div>
  );
}
