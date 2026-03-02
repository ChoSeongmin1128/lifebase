export interface CalendarData {
  id: string;
  name: string;
  color_id: string | null;
  is_primary: boolean;
  is_visible: boolean;
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
