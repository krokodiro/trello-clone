"use client";

import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { getAccessToken, getWsUrl } from "@/lib/api";
import type { BoardDetail, Task, WSEvent } from "@/lib/types";

export function useBoardWebSocket(
  boardId: string,
  clientId: string,
  enabled = true
) {
  const queryClient = useQueryClient();
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!enabled || !boardId) return;

    const token = getAccessToken();
    if (!token) return;

    const ws = new WebSocket(
      `${getWsUrl()}/ws?board_id=${boardId}&token=${encodeURIComponent(token)}`
    );
    wsRef.current = ws;

    ws.onmessage = (event) => {
      const data: WSEvent = JSON.parse(event.data);
      if (data.client_id && data.client_id === clientId) return;

      queryClient.setQueryData<BoardDetail>(["board", boardId], (old) => {
        if (!old) return old;
        return applyWSEvent(old, data);
      });

      if (
        data.type === "task.updated" ||
        data.type === "comment.created"
      ) {
        const task = data.payload as Task;
        if (task?.id) {
          queryClient.invalidateQueries({ queryKey: ["task", task.id] });
        }
      }
    };

    return () => {
      ws.close();
      wsRef.current = null;
    };
  }, [boardId, clientId, enabled, queryClient]);
}

function applyWSEvent(board: BoardDetail, event: WSEvent): BoardDetail {
  const lists = board.lists.map((l) => ({
    ...l,
    tasks: [...(l.tasks || [])],
  }));

  switch (event.type) {
    case "task.created": {
      const task = event.payload as Task;
      const list = lists.find((l) => l.id === task.list_id);
      if (list) list.tasks = [...(list.tasks || []), task];
      break;
    }
    case "task.updated":
    case "task.moved": {
      const task = event.payload as Task;
      for (const list of lists) {
        list.tasks = (list.tasks || []).filter((t) => t.id !== task.id);
      }
      const target = lists.find((l) => l.id === task.list_id);
      if (target) {
        const tasks = [...(target.tasks || [])];
        tasks.splice(task.position, 0, task);
        target.tasks = tasks.map((t, i) => ({ ...t, position: i }));
      }
      break;
    }
    case "task.deleted": {
      const { id } = event.payload as { id: string };
      for (const list of lists) {
        list.tasks = (list.tasks || []).filter((t) => t.id !== id);
      }
      break;
    }
    case "list.created": {
      const list = event.payload as (typeof lists)[0];
      lists.push({ ...list, tasks: [] });
      break;
    }
    case "list.updated": {
      const list = event.payload as (typeof lists)[0];
      const idx = lists.findIndex((l) => l.id === list.id);
      if (idx >= 0) lists[idx] = { ...lists[idx], ...list };
      break;
    }
    case "list.deleted": {
      const { id } = event.payload as { id: string };
      return { ...board, lists: lists.filter((l) => l.id !== id) };
    }
    case "task.assignee.added":
    case "task.assignee.removed":
      return board;
    default:
      break;
  }

  return { ...board, lists };
}
