"use client";

import { useState } from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { mutationError } from "@/lib/toast-utils";
import type { Workspace } from "@/lib/types";
import { AuthGuard } from "@/components/auth-guard";
import { Button, Card, EmptyState, Input, PageHeader, PageLoader } from "@/components/ui";
import { useAuth } from "@/providers/auth-provider";
import { useToast } from "@/providers/toast-provider";

const ACCENTS = [
  "from-blue-600 to-blue-500",
  "from-violet-600 to-violet-500",
  "from-emerald-600 to-emerald-500",
  "from-amber-600 to-amber-500",
  "from-rose-600 to-rose-500",
  "from-cyan-600 to-cyan-500",
];

export default function HomePage() {
  return (
    <AuthGuard>
      <WorkspacePicker />
    </AuthGuard>
  );
}

function WorkspacePicker() {
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const { user } = useAuth();
  const [name, setName] = useState("");

  const { data: workspaces, isLoading } = useQuery({
    queryKey: ["workspaces"],
    queryFn: () => api<Workspace[]>("/workspaces"),
  });

  const createWorkspace = useMutation({
    mutationFn: () =>
      api<Workspace>("/workspaces", {
        method: "POST",
        body: JSON.stringify({ name }),
      }),
    onSuccess: (ws) => {
      setName("");
      queryClient.invalidateQueries({ queryKey: ["workspaces"] });
      toast({ title: "Workspace created", description: ws.name, variant: "success" });
    },
    onError: (err) => {
      toast({ title: "Failed to create workspace", description: mutationError(err), variant: "error" });
    },
  });

  return (
    <div className="mx-auto max-w-5xl px-4 py-8 sm:px-6">
      <PageHeader
        title={user?.is_admin ? "All workspaces" : "Your workspaces"}
        subtitle={
          user?.is_admin
            ? "Platform admin — you can view and manage every workspace."
            : "Pick a workspace or create a new one to get started."
        }
      />

      {isLoading ? (
        <PageLoader label="Loading workspaces..." />
      ) : (workspaces || []).length === 0 ? (
        <EmptyState
          title="No workspaces yet"
          description="Create your first workspace below."
        />
      ) : (
        <div className="mb-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {(workspaces || []).map((ws, i) => (
            <Link key={ws.id} href={`/w/${ws.slug}`}>
              <Card hover className="group overflow-hidden p-0">
                <div
                  className={`h-20 bg-gradient-to-br ${ACCENTS[i % ACCENTS.length]}`}
                />
                <div className="p-4">
                  <h2 className="font-semibold text-foreground group-hover:text-primary">
                    {ws.name}
                  </h2>
                  <p className="text-sm text-muted">/{ws.slug}</p>
                </div>
              </Card>
            </Link>
          ))}
        </div>
      )}

      <Card>
        <h2 className="mb-1 font-semibold text-foreground">Create workspace</h2>
        <p className="mb-4 text-sm text-muted">
          Workspaces hold your boards and team members.
        </p>
        <form
          className="flex flex-col gap-3 sm:flex-row"
          onSubmit={(e) => {
            e.preventDefault();
            if (name.trim()) createWorkspace.mutate();
          }}
        >
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Marketing team"
          />
          <Button type="submit" disabled={!name.trim()} className="shrink-0" loading={createWorkspace.isPending}>
            Create
          </Button>
        </form>
      </Card>
    </div>
  );
}
