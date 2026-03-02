export type CalendarEvent = {
  id: string;
  title: string;
  start_time: string;
  end_time: string;
  is_all_day: boolean;
  color?: string;
};

export type SettingsResponse = {
  settings: Record<string, string>;
};
