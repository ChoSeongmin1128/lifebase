export type MobileTodoItem = {
  id: string;
  title: string;
  done: boolean;
  priority: string;
  due_date?: string;
  is_pinned: boolean;
};

export type MobileTodoList = {
  id: string;
  name: string;
};
