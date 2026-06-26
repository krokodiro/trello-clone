"use client";

import { Suspense, use, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { api } from "@/lib/api";
import { getRedirectPath } from "@/lib/redirect";
import { useAuth } from "@/providers/auth-provider";
import { Alert, AuthShell, Button, Spinner } from "@/components/ui";

export default function VerifyEmailTokenPage({
  params,
}: {
  params: Promise<{ token: string }>;
}) {
  return (
    <Suspense
      fallback={
        <AuthShell title="Email verification">
          <div className="flex flex-col items-center gap-3 py-4">
            <Spinner />
            <p className="text-sm text-muted">Loading...</p>
          </div>
        </AuthShell>
      }
    >
      <VerifyEmailTokenContent params={params} />
    </Suspense>
  );
}

function VerifyEmailTokenContent({
  params,
}: {
  params: Promise<{ token: string }>;
}) {
  const { token } = use(params);
  const { refreshUser, user } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();
  const redirect = getRedirectPath(searchParams.toString());
  const [status, setStatus] = useState<"loading" | "success" | "error">("loading");
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        await api("/auth/verify-email", {
          method: "POST",
          body: JSON.stringify({ token }),
        });
        if (cancelled) return;
        setStatus("success");
        if (user) {
          await refreshUser();
          setTimeout(() => router.push(redirect ?? "/"), 2000);
        }
      } catch (err) {
        if (cancelled) return;
        setStatus("error");
        setError(err instanceof Error ? err.message : "Verification failed");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [token, user, refreshUser, router, redirect]);

  return (
    <AuthShell title="Email verification">
      {status === "loading" && (
        <div className="flex flex-col items-center gap-3 py-4">
          <Spinner />
          <p className="text-sm text-muted">Verifying your email...</p>
        </div>
      )}
      {status === "success" && (
        <div className="space-y-4 text-center">
          <Alert variant="success">Your email has been verified.</Alert>
          <p className="text-sm text-muted">
            {user ? "Redirecting to your workspaces..." : "You can now sign in."}
          </p>
          {!user && (
            <Link href="/login">
              <Button className="w-full">Sign in</Button>
            </Link>
          )}
        </div>
      )}
      {status === "error" && (
        <div className="space-y-4 text-center">
          <Alert variant="error">{error}</Alert>
          <Link href="/verify-email">
            <Button className="w-full">Request a new link</Button>
          </Link>
        </div>
      )}
    </AuthShell>
  );
}
