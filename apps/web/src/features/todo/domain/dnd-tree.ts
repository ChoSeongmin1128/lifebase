export interface TodoItem {
  id: string;
  list_id: string;
  parent_id: string | null;
  title: string;
  notes: string;
  due_date: string | null;
  due_time: string | null;
  priority: string;
  is_done: boolean;
  is_pinned: boolean;
  starred_at?: string | null;
  sort_order: number;
  done_at: string | null;
  created_at: string;
  updated_at?: string;
}

export interface TodoNode extends TodoItem {
  children: TodoNode[];
}

export interface FlattenedItem {
  id: string;
  parentId: string | null;
  depth: number;
  index: number;
  todo: TodoNode;
}

export interface Projection {
  depth: number;
  parentId: string | null;
}

export interface ReorderChange {
  id: string;
  parent_id: string | null;
  sort_order: number;
}

export function buildTree(todos: TodoItem[]): TodoNode[] {
  const map = new Map<string, TodoNode>();
  const roots: TodoNode[] = [];

  for (const todo of todos) {
    map.set(todo.id, { ...todo, children: [] });
  }

  for (const todo of todos) {
    const node = map.get(todo.id)!;
    if (todo.parent_id && map.has(todo.parent_id)) {
      map.get(todo.parent_id)!.children.push(node);
    } else {
      roots.push(node);
    }
  }

  return roots;
}

/**
 * Flatten tree to a list with depth info.
 * When activeId is provided and that item has children, the children are excluded
 * (they move with the parent during drag).
 */
export function flattenTree(
  nodes: TodoNode[],
  collapsed: Set<string>,
  activeId?: string | null,
): FlattenedItem[] {
  const result: FlattenedItem[] = [];
  let index = 0;

  function walk(items: TodoNode[], depth: number) {
    for (const node of items) {
      result.push({ id: node.id, parentId: node.parent_id, depth, index: index++, todo: node });
      // Skip children of the dragged item (they move with parent)
      const isActive = activeId != null && node.id === activeId;
      if (node.children.length > 0 && !collapsed.has(node.id) && !isActive) {
        walk(node.children, depth + 1);
      }
    }
  }

  walk(nodes, 0);
  return result;
}

/**
 * Core projection algorithm: determine where the dragged item would land
 * based on vertical position (overId) and horizontal offset (dragOffsetX).
 */
export function getProjection(
  items: FlattenedItem[],
  activeId: string,
  overId: string,
  dragOffsetX: number,
  indentWidth: number = 24,
): Projection {
  const overIndex = items.findIndex((item) => item.id === overId);
  const activeIndex = items.findIndex((item) => item.id === activeId);
  if (overIndex === -1 || activeIndex === -1) {
    return { depth: 0, parentId: null };
  }

  // Items without the active item
  const itemsWithoutActive = items.filter((item) => item.id !== activeId);
  const newIndex = overIndex > activeIndex ? overIndex - 1 : overIndex;

  const prevItem = itemsWithoutActive[newIndex];
  const nextItem = itemsWithoutActive[newIndex + 1];

  const dragDepth = Math.round(dragOffsetX / indentWidth);
  const maxDepth = Math.min(prevItem ? prevItem.depth + 1 : 0, 1); // Max 1 level of nesting
  const minDepth = nextItem ? nextItem.depth : 0;

  let projectedDepth = prevItem ? prevItem.depth + dragDepth : 0;
  projectedDepth = Math.max(minDepth, Math.min(maxDepth, projectedDepth));

  // Determine parentId by walking backward to find the parent at projectedDepth - 1
  let parentId: string | null = null;
  if (projectedDepth > 0 && prevItem) {
    // Walk backward from the drop position to find the parent
    for (let i = newIndex; i >= 0; i--) {
      const item = itemsWithoutActive[i];
      if (item.depth === projectedDepth - 1) {
        parentId = item.id;
        break;
      }
    }
  }

  return { depth: projectedDepth, parentId };
}

/**
 * After a drag ends, compute the reorder changes (id, parentId, sortOrder)
 * for all items that changed.
 */
export function computeReorderChanges(
  items: FlattenedItem[],
  activeId: string,
  overId: string,
  projection: Projection,
): { updatedItems: FlattenedItem[]; changes: ReorderChange[] } {
  const activeIndex = items.findIndex((item) => item.id === activeId);
  const overIndex = items.findIndex((item) => item.id === overId);
  if (activeIndex === -1 || overIndex === -1) {
    return { updatedItems: items, changes: [] };
  }

  // Get the active item and its children (for parent drag)
  const activeItem = items[activeIndex];
  const activeChildren: FlattenedItem[] = [];
  for (let i = activeIndex + 1; i < items.length; i++) {
    if (items[i].depth > activeItem.depth) {
      activeChildren.push(items[i]);
    } else {
      break;
    }
  }

  // Remove active + its children from the list
  const idsToRemove = new Set([activeItem.id, ...activeChildren.map((c) => c.id)]);
  const remaining = items.filter((item) => !idsToRemove.has(item.id));

  // Find new insertion point
  let insertIndex = remaining.findIndex((item) => item.id === overId);
  if (insertIndex === -1) insertIndex = remaining.length;
  // If we moved down, insert after the over item
  if (overIndex > activeIndex) insertIndex++;

  // Build moved items with new depth/parentId
  const movedActive: FlattenedItem = {
    ...activeItem,
    depth: projection.depth,
    parentId: projection.parentId,
  };
  const depthDiff = projection.depth - activeItem.depth;
  const movedChildren = activeChildren.map((child) => ({
    ...child,
    depth: child.depth + depthDiff,
    parentId: child.parentId === activeItem.parentId ? projection.parentId : child.parentId,
  }));

  // Insert moved items
  const updatedItems = [
    ...remaining.slice(0, insertIndex),
    movedActive,
    ...movedChildren,
    ...remaining.slice(insertIndex),
  ];

  // Compute sort_order per parent group and detect changes
  const changes: ReorderChange[] = [];
  const parentCounters = new Map<string, number>();
  const parentKey = (parentId: string | null) => parentId ?? "__root__";

  for (const item of updatedItems) {
    const key = parentKey(item.parentId);
    const sortOrder = parentCounters.get(key) ?? 0;
    parentCounters.set(key, sortOrder + 1);

    // Find the original item
    const original = items.find((i) => i.id === item.id);
    const newParentId = item.id === activeId ? projection.parentId : item.parentId;
    if (
      !original ||
      original.todo.sort_order !== sortOrder ||
      (original.parentId ?? null) !== (newParentId ?? null)
    ) {
      changes.push({
        id: item.id,
        parent_id: newParentId,
        sort_order: sortOrder,
      });
    }
  }

  return { updatedItems, changes };
}
