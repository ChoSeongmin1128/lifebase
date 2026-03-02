"use client";

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";
import { useThemeContext } from "@/components/providers/ThemeProvider";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from "@/components/ui/select";
import { Sun, Moon, Monitor } from "lucide-react";

export default function SettingsPage() {
  const [settings, setSettings] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(true);
  const { theme, setTheme } = useThemeContext();

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
  const dndEnabled = get("dnd_enabled", "true") === "true";

  const handleThemeChange = (value: "light" | "dark" | "system") => {
    setTheme(value);
    updateSetting("theme", value);
  };

  return (
    <Tabs defaultValue="general" className="flex h-full flex-col">
      <TabsList className="px-4 md:px-6">
        <TabsTrigger value="general">일반</TabsTrigger>
        <TabsTrigger value="calendar">캘린더</TabsTrigger>
        <TabsTrigger value="todo">Todo</TabsTrigger>
        <TabsTrigger value="notification">알림</TabsTrigger>
        <TabsTrigger value="cloud">Cloud</TabsTrigger>
      </TabsList>

      <div className="flex-1 overflow-auto p-4 md:p-6">
        {loading ? (
          <div className="flex items-center justify-center py-20 text-text-muted">
            불러오는 중...
          </div>
        ) : (
          <div className="mx-auto max-w-2xl space-y-6">
            {/* General */}
            <TabsContent value="general" className="space-y-6">
              <SettingsCard title="테마">
                <SettingRow label="모드">
                  <div className="flex gap-1">
                    {([
                      { value: "light" as const, icon: Sun, label: "라이트" },
                      { value: "dark" as const, icon: Moon, label: "다크" },
                      { value: "system" as const, icon: Monitor, label: "시스템" },
                    ]).map(({ value, icon: Icon, label }) => (
                      <Button
                        key={value}
                        variant={theme === value ? "primary" : "ghost"}
                        size="sm"
                        onClick={() => handleThemeChange(value)}
                        className="gap-1.5"
                      >
                        <Icon size={14} />
                        {label}
                      </Button>
                    ))}
                  </div>
                </SettingRow>
              </SettingsCard>

              <SettingsCard title="계정">
                <SettingRow label="Google 계정">
                  <span className="text-sm text-text-muted">연결됨</span>
                </SettingRow>
              </SettingsCard>
            </TabsContent>

            {/* Calendar */}
            <TabsContent value="calendar" className="space-y-6">
              <SettingsCard title="캘린더 표시">
                <SettingRow label="기본 뷰">
                  <Select
                    value={get("calendar_default_view", "month")}
                    onValueChange={(v) => updateSetting("calendar_default_view", v)}
                  >
                    <SelectTrigger className="w-32 h-8 text-xs">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="month">월간</SelectItem>
                      <SelectItem value="week">주간</SelectItem>
                      <SelectItem value="3day">3일</SelectItem>
                      <SelectItem value="agenda">일정</SelectItem>
                      <SelectItem value="year-compact">연간 컴팩트</SelectItem>
                      <SelectItem value="year-timeline">연간 타임라인</SelectItem>
                    </SelectContent>
                  </Select>
                </SettingRow>
                <Separator />
                <SettingRow label="주 시작 요일">
                  <Select
                    value={get("calendar_week_start", "0")}
                    onValueChange={(v) => updateSetting("calendar_week_start", v)}
                  >
                    <SelectTrigger className="w-28 h-8 text-xs">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="0">일요일</SelectItem>
                      <SelectItem value="1">월요일</SelectItem>
                      <SelectItem value="2">화요일</SelectItem>
                      <SelectItem value="3">수요일</SelectItem>
                      <SelectItem value="4">목요일</SelectItem>
                      <SelectItem value="5">금요일</SelectItem>
                      <SelectItem value="6">토요일</SelectItem>
                    </SelectContent>
                  </Select>
                </SettingRow>
                <Separator />
                <SettingRow label="주간 뷰 시작 시간">
                  <Select
                    value={get("week_start_hour", "8")}
                    onValueChange={(v) => updateSetting("week_start_hour", v)}
                  >
                    <SelectTrigger className="w-24 h-8 text-xs">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {Array.from({ length: 24 }, (_, i) => (
                        <SelectItem key={i} value={String(i)}>
                          {String(i).padStart(2, "0")}:00
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </SettingRow>
                <Separator />
                <SettingRow label="주간 뷰 종료 시간">
                  <Select
                    value={get("week_end_hour", "22")}
                    onValueChange={(v) => updateSetting("week_end_hour", v)}
                  >
                    <SelectTrigger className="w-24 h-8 text-xs">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {Array.from({ length: 25 }, (_, i) => (
                        <SelectItem key={i} value={String(i)}>
                          {String(i).padStart(2, "0")}:00
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </SettingRow>
              </SettingsCard>
            </TabsContent>

            {/* Todo */}
            <TabsContent value="todo" className="space-y-6">
              <SettingsCard title="Todo 설정">
                <SettingRow label="기본 정렬">
                  <Select
                    value={get("todo_default_sort", "due")}
                    onValueChange={(v) => updateSetting("todo_default_sort", v)}
                  >
                    <SelectTrigger className="w-32 h-8 text-xs">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="due">마감일순</SelectItem>
                      <SelectItem value="priority">우선순위순</SelectItem>
                      <SelectItem value="created_at">생성일순</SelectItem>
                      <SelectItem value="manual">수동 정렬</SelectItem>
                    </SelectContent>
                  </Select>
                </SettingRow>
                <Separator />
                <SettingRow label="캘린더에 완료된 Todo 표시">
                  <div className="flex gap-1">
                    {[
                      { value: "true", label: "표시" },
                      { value: "false", label: "숨김" },
                    ].map((opt) => (
                      <Button
                        key={opt.value}
                        variant={get("show_done_in_calendar", "false") === opt.value ? "primary" : "ghost"}
                        size="sm"
                        onClick={() => updateSetting("show_done_in_calendar", opt.value)}
                      >
                        {opt.label}
                      </Button>
                    ))}
                  </div>
                </SettingRow>
              </SettingsCard>
            </TabsContent>

            {/* Notification */}
            <TabsContent value="notification" className="space-y-6">
              <SettingsCard title="알림">
                <SettingRow label="Push 알림">
                  <div className="flex gap-1">
                    {[
                      { value: "true", label: "켜짐" },
                      { value: "false", label: "꺼짐" },
                    ].map((opt) => (
                      <Button
                        key={opt.value}
                        variant={get("push_enabled", "true") === opt.value ? "primary" : "ghost"}
                        size="sm"
                        onClick={() => updateSetting("push_enabled", opt.value)}
                      >
                        {opt.label}
                      </Button>
                    ))}
                  </div>
                </SettingRow>
                <Separator />
                <SettingRow label="방해 금지">
                  <div className="flex gap-1">
                    {[
                      { value: "true", label: "켜짐" },
                      { value: "false", label: "꺼짐" },
                    ].map((opt) => (
                      <Button
                        key={opt.value}
                        variant={get("dnd_enabled", "true") === opt.value ? "primary" : "ghost"}
                        size="sm"
                        onClick={() => updateSetting("dnd_enabled", opt.value)}
                      >
                        {opt.label}
                      </Button>
                    ))}
                  </div>
                </SettingRow>
                <Separator />
                <SettingRow label="방해 금지 시작">
                  <Select
                    value={get("dnd_start", "23")}
                    onValueChange={(v) => updateSetting("dnd_start", v)}
                  >
                    <SelectTrigger
                      disabled={!dndEnabled}
                      className={`h-8 w-24 text-xs ${!dndEnabled ? "opacity-50" : ""}`}
                    >
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {Array.from({ length: 24 }, (_, i) => (
                        <SelectItem key={i} value={String(i)}>
                          {String(i).padStart(2, "0")}:00
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </SettingRow>
                <Separator />
                <SettingRow label="방해 금지 종료">
                  <Select
                    value={get("dnd_end", "7")}
                    onValueChange={(v) => updateSetting("dnd_end", v)}
                  >
                    <SelectTrigger
                      disabled={!dndEnabled}
                      className={`h-8 w-24 text-xs ${!dndEnabled ? "opacity-50" : ""}`}
                    >
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {Array.from({ length: 25 }, (_, i) => (
                        <SelectItem key={i} value={String(i)}>
                          {String(i).padStart(2, "0")}:00
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </SettingRow>
                <Separator />
                <SettingRow label="긴급만 허용">
                  <div className="flex gap-1">
                    {[
                      { value: "true", label: "켜짐" },
                      { value: "false", label: "꺼짐" },
                    ].map((opt) => (
                      <Button
                        key={opt.value}
                        variant={get("dnd_urgent_only", "false") === opt.value ? "primary" : "ghost"}
                        size="sm"
                        disabled={!dndEnabled}
                        onClick={() => updateSetting("dnd_urgent_only", opt.value)}
                      >
                        {opt.label}
                      </Button>
                    ))}
                  </div>
                </SettingRow>
              </SettingsCard>
            </TabsContent>

            {/* Cloud */}
            <TabsContent value="cloud" className="space-y-6">
              <SettingsCard title="스토리지">
                <SettingRow label="사용량">
                  <span className="text-sm text-text-muted">계산 중...</span>
                </SettingRow>
                <Separator />
                <SettingRow label="할당량">
                  <span className="text-sm text-text-muted">1 TB</span>
                </SettingRow>
              </SettingsCard>
              <SettingsCard title="Cloud 정렬">
                <SettingRow label="기본 정렬">
                  <Select
                    value={get("cloud_default_sort", "name")}
                    onValueChange={(v) => updateSetting("cloud_default_sort", v)}
                  >
                    <SelectTrigger className="w-32 h-8 text-xs">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="name">이름순</SelectItem>
                      <SelectItem value="updated_at">수정일순</SelectItem>
                      <SelectItem value="created_at">생성일순</SelectItem>
                      <SelectItem value="size">크기순</SelectItem>
                    </SelectContent>
                  </Select>
                </SettingRow>
              </SettingsCard>
            </TabsContent>
          </div>
        )}
      </div>
    </Tabs>
  );
}

function SettingsCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-border p-4">
      <h3 className="mb-3 text-sm font-medium text-text-strong">{title}</h3>
      <div className="space-y-3">{children}</div>
    </div>
  );
}

function SettingRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-sm text-text-secondary">{label}</span>
      {children}
    </div>
  );
}
