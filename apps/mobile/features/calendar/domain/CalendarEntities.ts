export type CalendarEvent = {
  id: string;
  calendar_id: string;
  title: string;
  description?: string;
  location?: string;
  start_time: string;
  end_time: string;
  timezone?: string;
  is_all_day: boolean;
  color_id?: string | null;
};

export type CalendarData = {
  id: string;
  name: string;
  color_id: string | null;
  google_account_id: string | null;
  is_primary: boolean;
  is_visible: boolean;
  kind: "primary" | "custom" | "holiday" | "birthday" | "subscribed" | string;
  is_readonly: boolean;
  is_special: boolean;
  synced_start: string | null;
  synced_end: string | null;
};

export type SettingsResponse = {
  settings: Record<string, string>;
};

export type BackfillEventsResult = {
  fetched_events: number;
  updated_events: number;
  deleted_events: number;
  covered_start: string;
  covered_end: string;
};

export type DaySummaryHoliday = {
  date: string;
  name: string;
};

export type DaySummaryTodo = {
  id: string;
  list_id: string;
  title: string;
  notes?: string;
  due: string | null;
  due_date?: string | null;
  due_time?: string | null;
  priority: string;
  is_done: boolean;
};

export type DaySummaryData = {
  date: string;
  timezone: string;
  holidays: DaySummaryHoliday[];
  events: CalendarEvent[];
  todos: DaySummaryTodo[];
};
