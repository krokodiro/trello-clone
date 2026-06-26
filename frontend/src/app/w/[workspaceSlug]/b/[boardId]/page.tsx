"use client";

import { use, useState } from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { mutationError } from "@/lib/toast-utils";
import type { BoardDetail, Workspace, WorkspaceMember } from "@/lib/types";
import { AuthGuard } from "@/components/auth-guard";
import { BoardCanvas } from "@/components/board-canvas";
import { Button, Input, Spinner } from "@/components/ui";
import { useToast } from "@/providers/toast-provider";

export default function BoardPage({
  params,
}: {
  params: Promise<{ workspaceSlug: string; boardId: string }>;
}) {
  const { workspaceSlug, boardId } = use(params);
  return (
    <AuthGuard>
      <BoardContent workspaceSlug={workspaceSlug} boardId={boardId} />
    </AuthGuard>
  );
}

function BoardContent({
  workspaceSlug,
  boardId,
}: {
  workspaceSlug: string;
  boardId: string;
}) {
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const [listName, setListName] = useState("");

  const { data: workspaces } = useQuery({
    queryKey: ["workspaces"],
    queryFn: () => api<Workspace[]>("/workspaces"),
  });
  const workspace = workspaces?.find((w) => w.slug === workspaceSlug);

  const { data: board, isLoading, isError, error } = useQuery({
    queryKey: ["board", boardId],
    queryFn: () => api<BoardDetail>(`/boards/${boardId}`),
  });

  const { data: members } = useQuery({
    queryKey: ["members", workspace?.id],
    queryFn: () => api<WorkspaceMember[]>(`/workspaces/${workspace!.id}/members`),
    enabled: !!workspace?.id,
  });

  const createList = useMutation({
    mutationFn: () =>
      api(`/boards/${boardId}/lists`, {
        method: "POST",
        body: JSON.stringify({ name: listName }),
      }),
    onSuccess: () => {
      setListName("");
      queryClient.invalidateQueries({ queryKey: ["board", boardId] });
      toast({ title: "List added", variant: "success" });
    },
    onError: (err) => {
      toast({ title: "Failed to add list", description: mutationError(err), variant: "error" });
    },
  });

  if (isLoading) {
    return (
      <div className="board-bg flex min-h-[calc(100vh-3.5rem)] items-center justify-center">
        <div className="flex items-center gap-3 rounded-lg bg-white/10 px-4 py-3 text-white backdrop-blur-sm">
          <Spinner className="border-white/30 border-t-white" />
          <span className="text-sm">Loading board...</span>
        </div>
      </div>
    );
  }

  if (isError || !board) {
    return (
      <div className="board-bg flex min-h-[calc(100vh-3.5rem)] items-center justify-center p-6">
        <div className="rounded-lg bg-surface p-6 text-center shadow-md">
          <p className="mb-4 text-foreground">
            {error instanceof Error ? error.message : "Board not found"}
          </p>
          <Link href={`/w/${workspaceSlug}`}>
            <Button variant="secondary">← Back to workspace</Button>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="board-bg min-h-[calc(100vh-3.5rem)] px-4 py-5 sm:px-6">
      <div className="mb-5 flex flex-wrap items-center justify-between gap-4">
        <div>
          <Link
            href={`/w/${workspaceSlug}`}
            className="text-sm text-white/80 transition-colors hover:text-white"
          >
            ← {workspace?.name || "Workspace"}
          </Link>
          <h1 className="text-xl font-bold text-white drop-shadow-sm">
            {board.board.name}
          </h1>
        </div>
        <form
          className="flex gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            if (listName.trim()) createList.mutate();
          }}
        >
          <Input
            value={listName}
            onChange={(e) => setListName(e.target.value)}
            placeholder="New list name"
            className="w-40 border-white/20 bg-white/95 sm:w-48"
          />
          <Button type="submit" variant="board" loading={createList.isPending}>
            Add list
          </Button>
        </form>
      </div>

      <BoardCanvas boardId={boardId} board={board} members={members || []} />
    </div>
  );
}
