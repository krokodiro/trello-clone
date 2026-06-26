"use client";

import { use, useState } from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { mutationError } from "@/lib/toast-utils";
import type { Board, Workspace, WorkspaceMember } from "@/lib/types";
import { AuthGuard } from "@/components/auth-guard";
import {
  Avatar,
  Badge,
  Button,
  Card,
  EmptyState,
  Input,
  PageHeader,
  PageLoader,
} from "@/components/ui";
import { useToast } from "@/providers/toast-provider";

const BOARD_ACCENTS = [
  "from-sky-600 to-blue-700",
  "from-indigo-600 to-violet-700",
  "from-teal-600 to-emerald-700",
  "from-orange-500 to-amber-600",
];

export default function WorkspacePage({
  params,
}: {
  params: Promise<{ workspaceSlug: string }>;
}) {
  const { workspaceSlug } = use(params);
  return (
    <AuthGuard>
      <WorkspaceContent slug={workspaceSlug} />
    </AuthGuard>
  );
}

function WorkspaceContent({ slug }: { slug: string }) {
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const [boardName, setBoardName] = useState("");
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteUrl, setInviteUrl] = useState("");

  const { data: workspaces } = useQuery({
    queryKey: ["workspaces"],
    queryFn: () => api<Workspace[]>("/workspaces"),
  });

  const workspace = workspaces?.find((w) => w.slug === slug);

  const { data: boards, isLoading } = useQuery({
    queryKey: ["boards", workspace?.id],
    queryFn: () => api<Board[]>(`/workspaces/${workspace!.id}/boards`),
    enabled: !!workspace?.id,
  });

  const { data: members } = useQuery({
    queryKey: ["members", workspace?.id],
    queryFn: () => api<WorkspaceMember[]>(`/workspaces/${workspace!.id}/members`),
    enabled: !!workspace?.id,
  });

  const createBoard = useMutation({
    mutationFn: () =>
      api<Board>(`/workspaces/${workspace!.id}/boards`, {
        method: "POST",
        body: JSON.stringify({ name: boardName }),
      }),
    onSuccess: (board) => {
      setBoardName("");
      queryClient.invalidateQueries({ queryKey: ["boards", workspace?.id] });
      toast({ title: "Board created", description: board.name, variant: "success" });
    },
    onError: (err) => {
      toast({ title: "Failed to create board", description: mutationError(err), variant: "error" });
    },
  });

  const invite = useMutation({
    mutationFn: () =>
      api<{ invite_url: string }>(`/workspaces/${workspace!.id}/invitations`, {
        method: "POST",
        body: JSON.stringify({ email: inviteEmail, role: "member" }),
      }),
    onSuccess: (data) => {
      setInviteUrl(data.invite_url);
      setInviteEmail("");
      toast({ title: "Invitation sent", description: "Share the invite link with your teammate.", variant: "success" });
    },
    onError: (err) => {
      toast({ title: "Failed to send invitation", description: mutationError(err), variant: "error" });
    },
  });

  if (!workspace) {
    return (
      <div className="mx-auto max-w-5xl px-4 py-8 sm:px-6">
        <EmptyState title="Workspace not found" />
        <Link href="/" className="mt-4 inline-block text-sm text-primary hover:underline">
          ← Back to workspaces
        </Link>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-5xl px-4 py-8 sm:px-6">
      <PageHeader
        backHref="/"
        backLabel="Workspaces"
        title={workspace.name}
        subtitle={`/${workspace.slug}`}
      />

      <h2 className="mb-4 text-sm font-semibold uppercase tracking-wide text-muted">
        Boards
      </h2>
      {isLoading ? (
        <PageLoader label="Loading boards..." />
      ) : (boards || []).length === 0 ? (
        <EmptyState
          title="No boards yet"
          description="Create your first board to start organizing tasks."
        />
      ) : (
        <div className="mb-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {(boards || []).map((board, i) => (
            <Link key={board.id} href={`/w/${slug}/b/${board.id}`}>
              <Card hover className="group overflow-hidden p-0">
                <div
                  className={`flex h-24 items-end bg-gradient-to-br p-4 ${BOARD_ACCENTS[i % BOARD_ACCENTS.length]}`}
                >
                  <h3 className="font-semibold text-white drop-shadow-sm">
                    {board.name}
                  </h3>
                </div>
              </Card>
            </Link>
          ))}
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <h3 className="mb-1 font-semibold text-foreground">Create board</h3>
          <p className="mb-4 text-sm text-muted">Add a new board to this workspace.</p>
          <form
            className="flex flex-col gap-3 sm:flex-row"
            onSubmit={(e) => {
              e.preventDefault();
              if (boardName.trim()) createBoard.mutate();
            }}
          >
            <Input
              value={boardName}
              onChange={(e) => setBoardName(e.target.value)}
              placeholder="Board name"
            />
            <Button type="submit" className="shrink-0" loading={createBoard.isPending}>
              Create
            </Button>
          </form>
        </Card>

        <Card>
          <h3 className="mb-1 font-semibold text-foreground">Invite member</h3>
          <p className="mb-4 text-sm text-muted">Send an invite link by email.</p>
          <form
            className="flex flex-col gap-3 sm:flex-row"
            onSubmit={(e) => {
              e.preventDefault();
              if (inviteEmail.trim()) invite.mutate();
            }}
          >
            <Input
              type="email"
              value={inviteEmail}
              onChange={(e) => setInviteEmail(e.target.value)}
              placeholder="email@example.com"
            />
            <Button type="submit" className="shrink-0" loading={invite.isPending}>
              Invite
            </Button>
          </form>
          {inviteUrl && (
            <p className="mt-3 break-all rounded-md bg-background px-3 py-2 text-xs text-muted">
              Share link: <span className="font-mono text-foreground">{inviteUrl}</span>
            </p>
          )}
        </Card>
      </div>

      <Card className="mt-6">
        <h3 className="mb-4 font-semibold text-foreground">Members</h3>
        <ul className="divide-y divide-border">
          {(members || []).map((m) => (
            <li
              key={m.user_id}
              className="flex items-center justify-between gap-3 py-3 first:pt-0 last:pb-0"
            >
              <div className="flex items-center gap-3">
                {m.user && <Avatar name={m.user.name} size="sm" />}
                <div>
                  <p className="text-sm font-medium text-foreground">{m.user?.name}</p>
                  <p className="text-xs text-muted">{m.user?.email}</p>
                </div>
              </div>
              <Badge>{m.role}</Badge>
            </li>
          ))}
        </ul>
      </Card>
    </div>
  );
}
