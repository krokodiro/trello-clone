"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  closestCorners,
  useDroppable,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from "@dnd-kit/core";
import {
  SortableContext,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { mutationError } from "@/lib/toast-utils";
import type { BoardDetail, List, Task, WorkspaceMember } from "@/lib/types";
import { useBoardWebSocket } from "@/hooks/useBoardWebSocket";
import { TaskDetailModal } from "./task-detail-modal";
import { Avatar, Button, Input } from "./ui";
import { useToast } from "@/providers/toast-provider";

const clientId =
  typeof crypto !== "undefined"
    ? crypto.randomUUID()
    : Math.random().toString(36).slice(2);

function TaskCard({
  task,
  onClick,
}: {
  task: Task;
  onClick: () => void;
}) {
  const draggedRef = useRef(false);
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } =
    useSortable({ id: task.id, data: { type: "task", task } });

  useEffect(() => {
    if (isDragging) draggedRef.current = true;
  }, [isDragging]);

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  const dueSoon =
    task.due_date &&
    new Date(task.due_date) < new Date(Date.now() + 86400000 * 2);

  const handleClick = () => {
    if (draggedRef.current) {
      draggedRef.current = false;
      return;
    }
    onClick();
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      onClick={handleClick}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onClick();
        }
      }}
      role="button"
      tabIndex={0}
      className={`group cursor-grab rounded-lg bg-surface p-3 text-left shadow-sm transition-shadow hover:shadow-md active:cursor-grabbing ${
        isDragging ? "opacity-40" : ""
      }`}
    >
      <p className="text-sm font-medium leading-snug text-foreground">
        {task.title}
      </p>
      {((task.assignees?.length ?? 0) > 0 || task.due_date) && (
        <div className="mt-2.5 flex items-center justify-between gap-2">
          <div className="flex -space-x-1.5">
            {(task.assignees ?? []).map((a) => (
              <Avatar key={a.id} name={a.name} size="sm" />
            ))}
          </div>
          {task.due_date && (
            <span
              className={`rounded px-1.5 py-0.5 text-[10px] font-medium ${
                dueSoon
                  ? "bg-red-100 text-[var(--danger)]"
                  : "bg-background text-muted"
              }`}
            >
              {new Date(task.due_date).toLocaleDateString(undefined, {
                month: "short",
                day: "numeric",
              })}
            </span>
          )}
        </div>
      )}
    </div>
  );
}

function ListColumn({
  list,
  onAddTask,
  onTaskClick,
  addingTask,
}: {
  list: List;
  onAddTask: (listId: string, title: string) => void;
  onTaskClick: (taskId: string) => void;
  addingTask?: boolean;
}) {
  const [title, setTitle] = useState("");
  const [adding, setAdding] = useState(false);
  const tasks = list.tasks || [];
  const taskIds = useMemo(() => tasks.map((t) => t.id), [tasks]);
  const { setNodeRef, isOver } = useDroppable({ id: list.id });

  return (
    <div
      ref={setNodeRef}
      className={`flex w-72 shrink-0 flex-col rounded-xl bg-[var(--column)] p-2.5 transition-colors ${
        isOver ? "ring-2 ring-white/50" : ""
      }`}
    >
      <div className="mb-2 flex items-center justify-between px-1">
        <h3 className="text-sm font-semibold text-foreground">{list.name}</h3>
        <span className="rounded-full bg-black/5 px-2 py-0.5 text-xs text-muted">
          {tasks.length}
        </span>
      </div>
      <SortableContext items={taskIds} strategy={verticalListSortingStrategy}>
        <div className="flex min-h-8 flex-col gap-2">
          {tasks.map((task) => (
            <TaskCard
              key={task.id}
              task={task}
              onClick={() => onTaskClick(task.id)}
            />
          ))}
        </div>
      </SortableContext>
      {adding ? (
        <form
          className="mt-2 space-y-2"
          onSubmit={(e) => {
            e.preventDefault();
            if (!title.trim()) return;
            onAddTask(list.id, title.trim());
            setTitle("");
            setAdding(false);
          }}
        >
          <Input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Card title"
            className="text-sm"
            autoFocus
          />
          <div className="flex gap-2">
            <Button type="submit" size="sm" loading={addingTask}>
              Add card
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => {
                setAdding(false);
                setTitle("");
              }}
            >
              Cancel
            </Button>
          </div>
        </form>
      ) : (
        <button
          type="button"
          onClick={() => setAdding(true)}
          className="mt-2 rounded-md px-2 py-2 text-left text-sm text-muted transition-colors hover:bg-black/5 hover:text-foreground"
        >
          + Add a card
        </button>
      )}
    </div>
  );
}

type Props = {
  boardId: string;
  board: BoardDetail;
  members: WorkspaceMember[];
};

export function BoardCanvas({ boardId, board, members }: Props) {
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const [activeTask, setActiveTask] = useState<Task | null>(null);
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);

  useBoardWebSocket(boardId, clientId);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } })
  );

  const createTask = useMutation({
    mutationFn: ({ listId, title }: { listId: string; title: string }) =>
      api<Task>(`/lists/${listId}/tasks`, {
        method: "POST",
        body: JSON.stringify({ title }),
      }),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["board", boardId] }),
    onError: (err) => {
      toast({ title: "Failed to add card", description: mutationError(err), variant: "error" });
    },
  });

  const moveTask = useMutation({
    mutationFn: ({
      taskId,
      listId,
      position,
    }: {
      taskId: string;
      listId: string;
      position: number;
    }) =>
      api<Task>(`/tasks/${taskId}/move`, {
        method: "PATCH",
        body: JSON.stringify({
          list_id: listId,
          position,
          client_id: clientId,
        }),
      }),
    onMutate: async ({ taskId, listId, position }) => {
      await queryClient.cancelQueries({ queryKey: ["board", boardId] });
      const prev = queryClient.getQueryData<BoardDetail>(["board", boardId]);
      if (!prev) return { prev };

      const lists = prev.lists.map((l) => ({
        ...l,
        tasks: [...(l.tasks || [])],
      }));
      let movedTask: Task | undefined;
      for (const list of lists) {
        const idx = list.tasks!.findIndex((t) => t.id === taskId);
        if (idx >= 0) {
          [movedTask] = list.tasks!.splice(idx, 1);
          break;
        }
      }
      if (movedTask) {
        const target = lists.find((l) => l.id === listId);
        if (target) {
          movedTask = { ...movedTask, list_id: listId, position };
          target.tasks!.splice(position, 0, movedTask);
          target.tasks = target.tasks!.map((t, i) => ({ ...t, position: i }));
        }
      }
      queryClient.setQueryData(["board", boardId], { ...prev, lists });
      return { prev };
    },
    onError: (err, _vars, ctx) => {
      if (ctx?.prev) queryClient.setQueryData(["board", boardId], ctx.prev);
      toast({ title: "Failed to move card", description: mutationError(err), variant: "error" });
    },
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: ["board", boardId] }),
  });

  const findContainer = (taskId: string) => {
    for (const list of board.lists) {
      if ((list.tasks || []).some((t) => t.id === taskId)) return list.id;
    }
    return null;
  };

  const handleDragStart = (event: DragStartEvent) => {
    const task = event.active.data.current?.task as Task | undefined;
    if (task) setActiveTask(task);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    setActiveTask(null);
    const { active, over } = event;
    if (!over) return;

    const activeId = active.id as string;
    const overId = over.id as string;

    let targetListId = findContainer(overId);
    let targetPosition = 0;

    const overList = board.lists.find((l) => l.id === overId);
    if (overList) {
      targetListId = overId;
      targetPosition = overList.tasks?.length ?? 0;
    } else if (targetListId) {
      const list = board.lists.find((l) => l.id === targetListId);
      targetPosition =
        list?.tasks?.findIndex((t) => t.id === overId) ?? 0;
      if (targetPosition < 0) targetPosition = list?.tasks?.length ?? 0;
    } else {
      return;
    }

    const sourceListId = findContainer(activeId);
    if (!sourceListId || !targetListId) return;

    const sourceList = board.lists.find((l) => l.id === sourceListId);
    const sourceIndex = sourceList?.tasks?.findIndex((t) => t.id === activeId) ?? -1;
    if (sourceListId === targetListId && sourceIndex === targetPosition) return;

    moveTask.mutate({
      taskId: activeId,
      listId: targetListId,
      position: targetPosition,
    });
  };

  return (
    <>
      <DndContext
        sensors={sensors}
        collisionDetection={closestCorners}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
      >
        <div className="flex gap-4 overflow-x-auto pb-4">
          {board.lists.map((list) => (
            <ListColumn
              key={list.id}
              list={list}
              onAddTask={(listId, title) =>
                createTask.mutate({ listId, title })
              }
              onTaskClick={setSelectedTaskId}
              addingTask={createTask.isPending}
            />
          ))}
        </div>
        <DragOverlay>
          {activeTask ? (
            <div className="w-72 rotate-2 rounded-lg bg-surface p-3 shadow-lg">
              <p className="text-sm font-medium text-foreground">{activeTask.title}</p>
            </div>
          ) : null}
        </DragOverlay>
      </DndContext>

      {selectedTaskId && (
        <TaskDetailModal
          taskId={selectedTaskId}
          boardId={boardId}
          members={members}
          onClose={() => setSelectedTaskId(null)}
        />
      )}
    </>
  );
}
