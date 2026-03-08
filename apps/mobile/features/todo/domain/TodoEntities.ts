export type MobileTodoItem = {
  id: string;
  list_id?: string;
  title: string;
  notes: string;
  done: boolean;
  is_done?: boolean;
  due?: string | null;
  due_date?: string | null;
  due_time?: string | null;
  is_pinned: boolean;
  starred_at?: string | null;
  sort_order?: number;
  created_at?: string;
  updated_at?: string;
};

export type MobileTodoList = {
  id: string;
  name: string;
  google_account_id?: string | null;
  google_account_email?: string | null;
  source?: "google" | "local" | string;
};
