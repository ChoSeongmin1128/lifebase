export interface HomeSummaryEvent {
  id: string;
  calendar_id: string;
  title: string;
  start_time: string;
  end_time: string;
  is_all_day: boolean;
  color_id: string | null;
}

export interface HomeSummaryTodo {
  id: string;
  list_id: string;
  title: string;
  due_date: string | null;
  priority: string;
  is_pinned: boolean;
}

export interface HomeSummaryRecentFile {
  id: string;
  folder_id: string | null;
  name: string;
  mime_type: string;
  size_bytes: number;
  thumb_status: string;
  updated_at: string;
}

export interface HomeStorageTypeUsage {
  type: "image" | "video" | "other";
  bytes: number;
  percent: number;
}

export interface HomeSummary {
  window: {
    start: string;
    end: string;
  };
  events: {
    items: HomeSummaryEvent[];
    total_count: number;
  };
  todos: {
    overdue: HomeSummaryTodo[];
    today: HomeSummaryTodo[];
    overdue_count: number;
    today_count: number;
  };
  files: {
    recent: HomeSummaryRecentFile[];
    total_count: number;
  };
  storage: {
    used_bytes: number;
    quota_bytes: number;
    usage_percent: number;
    breakdown: HomeStorageTypeUsage[];
  };
}

export interface GetHomeSummaryInput {
  start: string;
  end: string;
  event_limit?: number;
  todo_limit?: number;
  recent_limit?: number;
}
