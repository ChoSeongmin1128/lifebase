"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import { useSettingsActions } from "@/features/settings/ui/hooks/useSettingsActions";
import { useAuthFlow } from "@/features/auth/ui/hooks/useAuthFlow";
import type { GoogleAccountSummary } from "@/features/auth/domain/AuthSession";
import { useThemeContext } from "@/components/providers/ThemeProvider";
import { useToast } from "@/components/providers/ToastProvider";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Input } from "@/components/ui/input";
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from "@/components/ui/select";
import { Sun, Moon, Monitor } from "lucide-react";
import {
  ACCOUNT_COLOR_PALETTE,
  MULTI_ACCOUNT_FALLBACK_COLORS,
  buildGoogleAccountAliasSettingKey,
  buildGoogleAccountColorSettingKey,
  getGoogleAccountAlias,
  getGoogleAccountCustomColor,
  getGoogleAccountDisplayName,
  isPresetAccountColor,
  normalizeHexColor,
} from "@/lib/google-account-preferences";

export default function SettingsPage() {
  const [settings, setSettings] = useState<Record<string, string>>({});
  const [googleAccounts, setGoogleAccounts] = useState<GoogleAccountSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [googleAccountsLoading, setGoogleAccountsLoading] = useState(true);
  const [aliasDrafts, setAliasDrafts] = useState<Record<string, string>>({});
  const [colorDrafts, setColorDrafts] = useState<Record<string, string>>({});
  const { theme, setTheme } = useThemeContext();
  const toast = useToast();
  const { getSettings, updateSetting } = useSettingsActions();
  const { requestAuthUrl, listGoogleAccounts, syncGoogleAccount } = useAuthFlow();

  const loadSettings = useCallback(async () => {
    setLoading(true);
    try {
      const next = await getSettings();
      setSettings(next);
    } catch {
      setSettings({});
    } finally {
      setLoading(false);
    }
  }, [getSettings]);

  const loadGoogleAccounts = useCallback(async () => {
    setGoogleAccountsLoading(true);
    try {
      const accounts = await listGoogleAccounts();
      setGoogleAccounts(accounts);
    } catch {
      setGoogleAccounts([]);
    } finally {
      setGoogleAccountsLoading(false);
    }
  }, [listGoogleAccounts]);

  useEffect(() => {
    loadSettings();
  }, [loadSettings]);

  useEffect(() => {
    loadGoogleAccounts();
  }, [loadGoogleAccounts]);

  useEffect(() => {
    setAliasDrafts((prev) => {
      const next = { ...prev };
      for (const account of googleAccounts) {
        if (next[account.id] !== undefined) continue;
        next[account.id] = getGoogleAccountAlias(settings, account.id);
      }
      return next;
    });
    setColorDrafts((prev) => {
      const next = { ...prev };
      for (const [index, account] of googleAccounts.entries()) {
        if (next[account.id] !== undefined) continue;
        const fallbackColor = MULTI_ACCOUNT_FALLBACK_COLORS[index % MULTI_ACCOUNT_FALLBACK_COLORS.length];
        const stored = getGoogleAccountCustomColor(settings, account.id);
        next[account.id] = isPresetAccountColor(stored) ? stored || fallbackColor : fallbackColor;
      }
      return next;
    });
  }, [googleAccounts, settings]);

  const handleUpdateSetting = async (key: string, value: string) => {
    setSettings((prev) => ({ ...prev, [key]: value }));
    try {
      await updateSetting(key, value);
    } catch (err) {
      console.error("Update setting failed:", err);
    }
  };

  const get = (key: string, fallback: string = "") => settings[key] ?? fallback;
  const dndEnabled = get("dnd_enabled", "true") === "true";

  const accountDefaultColorByID = useMemo(
    () =>
      new Map(
        googleAccounts.map((account, index) => [
          account.id,
          MULTI_ACCOUNT_FALLBACK_COLORS[index % MULTI_ACCOUNT_FALLBACK_COLORS.length],
        ]),
      ),
    [googleAccounts]
  );

  const handleThemeChange = (value: "light" | "dark" | "system") => {
    setTheme(value);
    handleUpdateSetting("theme", value);
  };

  const activeGoogleAccountCount = useMemo(
    () => googleAccounts.filter((account) => account.status === "active").length,
    [googleAccounts]
  );

  const handleConnectGoogleAccount = async () => {
    try {
      const data = await requestAuthUrl("web");
      sessionStorage.setItem("oauth_state", data.state);
      sessionStorage.setItem("oauth_intent", "link_google_account");
      sessionStorage.setItem("oauth_return_path", "/settings");
      window.location.href = data.url;
    } catch {
      toast.error("Google 계정 연결 요청 실패", "잠시 후 다시 시도해 주세요.");
    }
  };

  const getAccountSyncEnabled = (accountID: string, type: "calendar" | "todo") =>
    get(`google_account_sync_${type}_${accountID}`, "true") === "true";

  const handleToggleAccountSync = async (
    account: GoogleAccountSummary,
    type: "calendar" | "todo",
    enabled: boolean,
  ) => {
    if (account.status !== "active") return;

    const settingKey = `google_account_sync_${type}_${account.id}`;
    const previous = get(settingKey, "true");
    const nextValue = enabled ? "true" : "false";
    const calendarEnabled = type === "calendar" ? enabled : getAccountSyncEnabled(account.id, "calendar");
    const todoEnabled = type === "todo" ? enabled : getAccountSyncEnabled(account.id, "todo");

    setSettings((prev) => ({ ...prev, [settingKey]: nextValue }));
    try {
      await updateSetting(settingKey, nextValue);
      if (enabled) {
        await syncGoogleAccount(account.id, {
          sync_calendar: calendarEnabled,
          sync_todo: todoEnabled,
        });
      }
    } catch {
      try {
        await updateSetting(settingKey, previous);
      } catch {
        // noop
      }
      setSettings((prev) => ({ ...prev, [settingKey]: previous }));
      toast.error("동기화 설정 변경 실패", "잠시 후 다시 시도해 주세요.");
    }
  };

  const handleSaveAccountAlias = async (account: GoogleAccountSummary) => {
    const settingKey = buildGoogleAccountAliasSettingKey(account.id);
    const previous = settings[settingKey] ?? "";
    const nextAlias = (aliasDrafts[account.id] ?? "").trim();

    setSettings((prev) => ({ ...prev, [settingKey]: nextAlias }));
    try {
      await updateSetting(settingKey, nextAlias);
      toast.success("계정 별명 저장 완료");
    } catch (err) {
      console.error("Save account alias failed:", err);
      setSettings((prev) => ({ ...prev, [settingKey]: previous }));
      toast.error("계정 별명 저장 실패", "잠시 후 다시 시도해 주세요.");
    }
  };

  const handleSaveAccountColor = async (account: GoogleAccountSummary) => {
    const settingKey = buildGoogleAccountColorSettingKey(account.id);
    const previous = settings[settingKey] ?? "";
    const fallbackColor = accountDefaultColorByID.get(account.id) || MULTI_ACCOUNT_FALLBACK_COLORS[0];
    const normalized = normalizeHexColor(colorDrafts[account.id]);
    const nextColor = isPresetAccountColor(normalized) ? normalized || fallbackColor : fallbackColor;

    setColorDrafts((prev) => ({ ...prev, [account.id]: nextColor }));
    setSettings((prev) => ({ ...prev, [settingKey]: nextColor }));
    try {
      await updateSetting(settingKey, nextColor);
      toast.success("계정 색상 저장 완료");
    } catch (err) {
      console.error("Save account color failed:", err);
      setSettings((prev) => ({ ...prev, [settingKey]: previous }));
      setColorDrafts((prev) => ({
        ...prev,
        [account.id]: normalizeHexColor(previous) || fallbackColor,
      }));
      toast.error("계정 색상 저장 실패", "잠시 후 다시 시도해 주세요.");
    }
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
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-text-muted">
                      활성 {activeGoogleAccountCount}개 / 총 {googleAccounts.length}개
                    </span>
                    <Button size="sm" variant="secondary" onClick={handleConnectGoogleAccount}>
                      계정 추가 연결
                    </Button>
                  </div>
                </SettingRow>
                <Separator />
                <div className="space-y-2">
                  {googleAccountsLoading ? (
                    <p className="text-sm text-text-muted">Google 계정 정보를 불러오는 중...</p>
                  ) : googleAccounts.length === 0 ? (
                    <p className="text-sm text-text-muted">연결된 Google 계정이 없습니다.</p>
                  ) : (
                    googleAccounts.map((account) => (
                      <div
                        key={account.id}
                        className="rounded-md border border-border/70 px-3 py-2"
                      >
                        {(() => {
                          const alias = getGoogleAccountAlias(settings, account.id);
                          const displayName = getGoogleAccountDisplayName(settings, account.id, account.google_email);
                          const aliasKey = buildGoogleAccountAliasSettingKey(account.id);
                          const colorKey = buildGoogleAccountColorSettingKey(account.id);
                          const fallbackColor = accountDefaultColorByID.get(account.id) || MULTI_ACCOUNT_FALLBACK_COLORS[0];
                          const effectiveColor =
                            (isPresetAccountColor(colorDrafts[account.id])
                              ? normalizeHexColor(colorDrafts[account.id])
                              : null) ||
                            (isPresetAccountColor(settings[colorKey])
                              ? normalizeHexColor(settings[colorKey])
                              : null) ||
                            fallbackColor;
                          return (
                            <>
                        <div className="flex items-center justify-between gap-3">
                          <div className="min-w-0">
                            <p className="truncate text-sm font-medium text-text-strong">{displayName}</p>
                            {alias ? (
                              <p className="truncate text-xs text-text-muted">{account.google_email}</p>
                            ) : null}
                            <p className="text-xs text-text-muted">
                              연결일: {new Date(account.connected_at).toLocaleDateString("ko-KR")}
                            </p>
                          </div>
                          <div className="flex items-center gap-2">
                            {account.is_primary ? <Badge variant="primary">기본</Badge> : null}
                            <Badge variant={getGoogleAccountStatusBadgeVariant(account.status)}>
                              {getGoogleAccountStatusLabel(account.status)}
                            </Badge>
                          </div>
                        </div>
                        <div className="mt-3 flex flex-wrap items-end gap-2">
                          <div className="min-w-[12rem] flex-1">
                            <p className="mb-1 text-[11px] text-text-muted">표시 별명</p>
                            <Input
                              value={aliasDrafts[account.id] ?? settings[aliasKey] ?? ""}
                              placeholder={account.google_email}
                              className="h-8"
                              onChange={(e) =>
                                setAliasDrafts((prev) => ({ ...prev, [account.id]: e.target.value }))
                              }
                            />
                          </div>
                          <Button
                            type="button"
                            size="sm"
                            variant="secondary"
                            className="h-8"
                            onClick={() => handleSaveAccountAlias(account)}
                          >
                            별명 저장
                          </Button>
                        </div>
                        <div className="mt-2 flex flex-wrap items-end gap-2">
                          <div className="min-w-[14rem]">
                            <p className="mb-1 text-[11px] text-text-muted">다중 계정 색상</p>
                            <div className="flex flex-wrap gap-1.5">
                              {ACCOUNT_COLOR_PALETTE.map((color) => {
                                const selected = effectiveColor === color;
                                return (
                                  <button
                                    key={color}
                                    type="button"
                                    aria-label={`계정 색상 ${color}`}
                                    title={color}
                                    className={`h-6 w-6 rounded-full border transition ${
                                      selected
                                        ? "border-text-strong ring-2 ring-primary/35"
                                        : "border-border hover:scale-105"
                                    }`}
                                    style={{ backgroundColor: color }}
                                    onClick={() =>
                                      setColorDrafts((prev) => ({ ...prev, [account.id]: color }))
                                    }
                                  />
                                );
                              })}
                            </div>
                          </div>
                          <Button
                            type="button"
                            size="sm"
                            variant="secondary"
                            className="h-8"
                            onClick={() => handleSaveAccountColor(account)}
                          >
                            색상 저장
                          </Button>
                        </div>
                        <div className="mt-3 flex flex-wrap items-center gap-2">
                          <SyncToggle
                            label="캘린더 동기화"
                            enabled={getAccountSyncEnabled(account.id, "calendar")}
                            disabled={account.status !== "active"}
                            onToggle={(next) => handleToggleAccountSync(account, "calendar", next)}
                          />
                          <SyncToggle
                            label="Todo 동기화"
                            enabled={getAccountSyncEnabled(account.id, "todo")}
                            disabled={account.status !== "active"}
                            onToggle={(next) => handleToggleAccountSync(account, "todo", next)}
                          />
                          {account.status !== "active" ? (
                            <span className="text-xs text-text-muted">비활성 계정은 동기화 설정을 변경할 수 없습니다.</span>
                          ) : null}
                        </div>
                            </>
                          );
                        })()}
                      </div>
                    ))
                  )}
                </div>
              </SettingsCard>
            </TabsContent>

            {/* Calendar */}
            <TabsContent value="calendar" className="space-y-6">
              <SettingsCard title="캘린더 표시">
                <SettingRow label="기본 뷰">
                  <Select
                    value={get("calendar_default_view", "month")}
                    onValueChange={(v) => handleUpdateSetting("calendar_default_view", v)}
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
                    onValueChange={(v) => handleUpdateSetting("calendar_week_start", v)}
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
                    onValueChange={(v) => handleUpdateSetting("week_start_hour", v)}
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
                    onValueChange={(v) => handleUpdateSetting("week_end_hour", v)}
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
                <Separator />
                <SettingRow label="공휴일 표시">
                  <div className="flex gap-1">
                    {[
                      { value: "true", label: "표시" },
                      { value: "false", label: "숨김" },
                    ].map((opt) => (
                      <Button
                        key={opt.value}
                        variant={get("calendar_show_public_holidays", "true") === opt.value ? "primary" : "ghost"}
                        size="sm"
                        onClick={() => handleUpdateSetting("calendar_show_public_holidays", opt.value)}
                      >
                        {opt.label}
                      </Button>
                    ))}
                  </div>
                </SettingRow>
              </SettingsCard>
            </TabsContent>

            {/* Todo */}
            <TabsContent value="todo" className="space-y-6">
              <SettingsCard title="Todo 설정">
                <SettingRow label="기본 정렬">
                  <Select
                    value={get("todo_default_sort", "due")}
                    onValueChange={(v) => handleUpdateSetting("todo_default_sort", v)}
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
                <SettingRow label="완료 항목 보존 기간">
                  <Select
                    value={get("todo_done_retention_period", "1y")}
                    onValueChange={(v) => handleUpdateSetting("todo_done_retention_period", v)}
                  >
                    <SelectTrigger className="w-32 h-8 text-xs">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="1m">1달</SelectItem>
                      <SelectItem value="3m">3달</SelectItem>
                      <SelectItem value="6m">반년</SelectItem>
                      <SelectItem value="1y">1년</SelectItem>
                      <SelectItem value="3y">3년</SelectItem>
                      <SelectItem value="unlimited">무제한</SelectItem>
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
                        onClick={() => handleUpdateSetting("show_done_in_calendar", opt.value)}
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
                        onClick={() => handleUpdateSetting("push_enabled", opt.value)}
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
                        onClick={() => handleUpdateSetting("dnd_enabled", opt.value)}
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
                    onValueChange={(v) => handleUpdateSetting("dnd_start", v)}
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
                    onValueChange={(v) => handleUpdateSetting("dnd_end", v)}
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
                        onClick={() => handleUpdateSetting("dnd_urgent_only", opt.value)}
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
                    onValueChange={(v) => handleUpdateSetting("cloud_default_sort", v)}
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

function SyncToggle({
  label,
  enabled,
  disabled,
  onToggle,
}: {
  label: string;
  enabled: boolean;
  disabled?: boolean;
  onToggle: (next: boolean) => void;
}) {
  return (
    <div className="inline-flex items-center gap-2 rounded-md border border-border/70 px-2 py-1">
      <span className="text-xs text-text-secondary">{label}</span>
      <Button
        type="button"
        size="sm"
        variant={enabled ? "primary" : "ghost"}
        className="h-6 px-2 text-xs"
        disabled={disabled}
        onClick={() => onToggle(!enabled)}
      >
        {enabled ? "켜짐" : "꺼짐"}
      </Button>
    </div>
  );
}

function getGoogleAccountStatusLabel(status: string): string {
  if (status === "active") return "정상";
  if (status === "reauth_required") return "재인증 필요";
  if (status === "revoked") return "해지됨";
  return status;
}

function getGoogleAccountStatusBadgeVariant(
  status: string
): "default" | "primary" | "error" | "warning" | "success" {
  if (status === "active") return "success";
  if (status === "reauth_required") return "warning";
  if (status === "revoked") return "error";
  return "default";
}
