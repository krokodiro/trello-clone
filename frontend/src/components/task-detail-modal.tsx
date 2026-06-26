"use client";

import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { mutationError } from "@/lib/toast-utils";
import type { Task, WorkspaceMember } from "@/lib/types";
import { Avatar, Button, FieldLabel, Input, Spinner, Textarea } from "./ui";
import { useToast } from "@/providers/toast-provider";

type Props = {
  taskId: string;
  boardId: string;
  members: WorkspaceMember[];
  onClose: () => void;
};

export function TaskDetailModal({
  taskId,
  boardId,
  members,
  onClose,
}: Props) {
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const { data: task, isLoading, isError } = useQuery({
    queryKey: ["task", taskId],
    queryFn: () => api<Task>(`/tasks/${taskId}`),
  });

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [dueDate, setDueDate] = useState("");
  const [comment, setComment] = useState("");

  useEffect(() => {
    if (task) {
      setTitle(task.title);
      setDescription(task.description || "");
      setDueDate(task.due_date ? task.due_date.slice(0, 10) : "");
    }
  }, [task]);

  const updateTask = useMutation({
    mutationFn: () =>
      api<Task>(`/tasks/${taskId}`, {
        method: "PATCH",
        body: JSON.stringify({
          title,
          description: description || null,
          due_date: dueDate ? new Date(dueDate).toISOString() : null,
        }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["board", boardId] });
      queryClient.invalidateQueries({ queryKey: ["task", taskId] });
      toast({ title: "Card saved", variant: "success" });
    },
    onError: (err) => {
      toast({ title: "Failed to save card", description: mutationError(err), variant: "error" });
    },
  });

  const addComment = useMutation({
    mutationFn: () =>
      api(`/tasks/${taskId}/comments`, {
        method: "POST",
        body: JSON.stringify({ body: comment }),
      }),
    onSuccess: () => {
      setComment("");
      queryClient.invalidateQueries({ queryKey: ["task", taskId] });
      toast({ title: "Comment posted", variant: "success" });
    },
    onError: (err) => {
      toast({ title: "Failed to post comment", description: mutationError(err), variant: "error" });
    },
  });

  const toggleAssignee = async (userId: string, has: boolean) => {
    try {
      if (has) {
        await api(`/tasks/${taskId}/assignees/${userId}`, { method: "DELETE" });
      } else {
        await api(`/tasks/${taskId}/assignees/${userId}`, { method: "POST" });
      }
      queryClient.invalidateQueries({ queryKey: ["task", taskId] });
      queryClient.invalidateQueries({ queryKey: ["board", boardId] });
    } catch (err) {
      toast({
        title: has ? "Failed to remove assignee" : "Failed to assign member",
        description: mutationError(err),
        variant: "error",
      });
    }
  };

  if (isLoading) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm">
        <div className="flex items-center gap-3 rounded-lg bg-surface px-6 py-4 shadow-lg">
          <Spinner />
          <span className="text-sm text-muted">Loading card...</span>
        </div>
      </div>
    );
  }

  if (!task) {
    return (
      <div
        className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm"
        onClick={onClose}
      >
        <div
          className="rounded-lg bg-surface p-6 shadow-lg"
          onClick={(e) => e.stopPropagation()}
        >
          <p className="mb-4 text-foreground">
            {isError ? "Failed to load card" : "Card not found"}
          </p>
          <Button variant="ghost" onClick={onClose}>
            Close
          </Button>
        </div>
      </div>
    );
  }

  const assigneeIds = new Set((task.assignees ?? []).map((a) => a.id));

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/40 p-4 pt-12 backdrop-blur-sm sm:pt-16"
      onClick={onClose}
    >
      <div
        className="w-full max-w-2xl rounded-xl bg-surface shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="border-b border-border px-6 py-4">
          <div className="flex items-start justify-between gap-4">
            <Input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="border-0 bg-transparent px-0 text-lg font-semibold shadow-none focus:ring-0"
            />
            <Button variant="ghost" size="sm" onClick={onClose}>
              ✕
            </Button>
          </div>
        </div>

        <div className="space-y-6 p-6">
          <div className="grid gap-6 md:grid-cols-2">
            <div>
              <FieldLabel>Description</FieldLabel>
              <Textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={4}
                placeholder="Add more details..."
              />
            </div>
            <div>
              <FieldLabel>Due date</FieldLabel>
              <Input
                type="date"
                value={dueDate}
                onChange={(e) => setDueDate(e.target.value)}
              />
            </div>
          </div>

          <div>
            <FieldLabel>Assignees</FieldLabel>
            <div className="flex flex-wrap gap-2">
              {(members ?? []).map((m) => {
                if (!m.user) return null;
                const has = assigneeIds.has(m.user.id);
                return (
                  <button
                    key={m.user.id}
                    type="button"
                    onClick={() => toggleAssignee(m.user!.id, has)}
                    className={`flex items-center gap-2 rounded-full border px-3 py-1.5 text-sm transition-colors ${
                      has
                        ? "border-primary bg-primary/10 text-primary"
                        : "border-border bg-surface text-muted hover:border-primary/30"
                    }`}
                  >
                    <Avatar name={m.user.name} size="sm" />
                    {m.user.name}
                  </button>
                );
              })}
            </div>
          </div>

          <Button
            onClick={() => updateTask.mutate()}
            disabled={updateTask.isPending}
            loading={updateTask.isPending}
          >
            Save changes
          </Button>

          <div className="border-t border-border pt-6">
            <h3 className="mb-4 text-sm font-semibold text-foreground">Comments</h3>
            <div className="mb-4 space-y-3">
              {(task.comments ?? []).length === 0 && (
                <p className="text-sm text-muted">No comments yet.</p>
              )}
              {(task.comments ?? []).map((c) => (
                <div
                  key={c.id}
                  className="flex gap-3 rounded-lg bg-background p-3"
                >
                  {c.user && <Avatar name={c.user.name} size="sm" />}
                  <div>
                    <p className="text-sm font-medium text-foreground">
                      {c.user?.name}
                    </p>
                    <p className="mt-0.5 text-sm text-muted">{c.body}</p>
                  </div>
                </div>
              ))}
            </div>
            <div className="flex gap-2">
              <Input
                value={comment}
                onChange={(e) => setComment(e.target.value)}
                placeholder="Write a comment..."
              />
              <Button
                onClick={() => addComment.mutate()}
                disabled={!comment.trim() || addComment.isPending}
                loading={addComment.isPending}
                className="shrink-0"
              >
                Post
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
