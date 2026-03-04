export interface CalendarData {
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
}

export interface EventData {
  id: string;
  calendar_id: string;
  title: string;
  description: string;
  location: string;
  start_time: string;
  end_time: string;
  timezone: string;
  is_all_day: boolean;
  color_id: string | null;
  recurrence_rule: string | null;
}

export interface CalendarSettingsResponse {
  settings: Record<string, string>;
}

export interface EventPayload {
  title: string;
  description: string;
  location: string;
  start_time: string;
  end_time: string;
  timezone: string;
  is_all_day: boolean;
}

export interface CreateEventInput extends EventPayload {
  calendar_id: string;
}

export interface BackfillEventsInput {
  start: string;
  end: string;
  calendar_ids?: string[];
  reason?: "range_backfill";
}

export interface BackfillEventsResult {
  fetched_events: number;
  updated_events: number;
  deleted_events: number;
  covered_start: string;
  covered_end: string;
}
