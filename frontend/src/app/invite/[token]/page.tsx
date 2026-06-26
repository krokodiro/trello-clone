"use client";

import { use, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { withRedirect } from "@/lib/redirect";
import type { Workspace } from "@/lib/types";
import { useAuth } from "@/providers/auth-provider";
import { Alert, AuthShell, Button, Spinner } from "@/components/ui";

export default function InvitePage({
  params,
}: {
  params: Promise<{ token: string }>;
}) {
  const { token } = use(params);
  const { user, loading } = useAuth();
  const router = useRouter();
  const [error, setError] = useState("");
  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [accepting, setAccepting] = useState(false);
  const invitePath = `/invite/${token}`;

  const accept = async () => {
    setAccepting(true);
    setError("");
    try {
      const ws = await api<Workspace>(`/invitations/${token}/accept`, {
        method: "POST",
      });
      setWorkspace(ws);
      const dest = `/w/${ws.slug}`;
      const next = user?.email_verified_at
        ? dest
        : withRedirect("/verify-email", dest);
      setTimeout(() => router.push(next), 1500);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to accept invitation");
    } finally {
      setAccepting(false);
    }
  };

  useEffect(() => {
    if (!loading && user && !workspace && !accepting && !error) {
      accept();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loading, user]);

  if (loading) {
    return (
      <div className="flex min-h-[calc(100vh-3.5rem)] flex-col items-center justify-center gap-3">
        <Spinner />
        <p className="text-sm text-muted">Loading...</p>
      </div>
    );
  }

  if (!user) {
    return (
      <AuthShell title="You're invited!" subtitle="Sign in to join this workspace">
        <div className="flex flex-col gap-2">
          <Link href={withRedirect("/login", invitePath)}>
            <Button className="w-full">Sign in</Button>
          </Link>
          <Link href={withRedirect("/register", invitePath)}>
            <Button variant="secondary" className="w-full">
              Create account
            </Button>
          </Link>
        </div>
      </AuthShell>
    );
  }

  return (
    <AuthShell title={workspace ? "Welcome!" : "Accept invitation"}>
      {workspace ? (
        <p className="text-center text-sm text-muted">
          You joined <strong className="text-foreground">{workspace.name}</strong>.
          {user.email_verified_at
            ? " Redirecting..."
            : " Verify your email to continue..."}
        </p>
      ) : (
        <div className="space-y-4 text-center">
          {error && <Alert variant="error">{error}</Alert>}
          {!user.email_verified_at && !error && (
            <p className="text-sm text-muted">
              Sign up with the same email address this invitation was sent to.
            </p>
          )}
          <Button onClick={accept} disabled={accepting} className="w-full">
            {accepting ? "Joining..." : "Join workspace"}
          </Button>
        </div>
      )}
    </AuthShell>
  );
}
