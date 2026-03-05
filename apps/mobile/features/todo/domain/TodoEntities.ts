export type MobileTodoItem = {
  id: string;
  list_id?: string;
  title: string;
  done: boolean;
  is_done?: boolean;
  priority: string;
  due?: string | null;
  due_date?: string;
  is_pinned: boolean;
};

export type MobileTodoList = {
  id: string;
  name: string;
  google_account_id?: string | null;
  google_account_email?: string | null;
  source?: "google" | "local" | string;
};
