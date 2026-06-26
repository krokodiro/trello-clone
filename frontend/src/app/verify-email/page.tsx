"use client";

import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useAuth } from "@/providers/auth-provider";
import { getRedirectPath } from "@/lib/redirect";
import { Alert, AuthShell, Button, Spinner } from "@/components/ui";

export default function VerifyEmailPage() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-[calc(100vh-3.5rem)] flex-col items-center justify-center gap-3">
          <Spinner />
          <p className="text-sm text-muted">Loading...</p>
        </div>
      }
    >
      <VerifyEmailContent />
    </Suspense>
  );
}

function VerifyEmailContent() {
  const { user, loading, verificationUrl, resendVerification, logout } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();
  const redirect = getRedirectPath(searchParams.toString());
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [sending, setSending] = useState(false);

  useEffect(() => {
    if (!loading && !user) {
      router.replace("/login");
      return;
    }
    if (!loading && user?.email_verified_at) {
      router.replace(redirect ?? "/");
    }
  }, [user, loading, router, redirect]);

  const handleResend = async () => {
    if (!user) return;
    setSending(true);
    setError("");
    setMessage("");
    try {
      await resendVerification(user.email);
      setMessage("Verification email sent. Check your inbox.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to send email");
    } finally {
      setSending(false);
    }
  };

  if (loading || !user || user.email_verified_at) {
    return (
      <div className="flex min-h-[calc(100vh-3.5rem)] flex-col items-center justify-center gap-3">
        <Spinner />
        <p className="text-sm text-muted">Loading...</p>
      </div>
    );
  }

  return (
    <AuthShell
      title="Verify your email"
      subtitle="Check your inbox for the verification link"
    >
      <p className="text-center text-sm text-muted">
        We sent a link to <strong className="text-foreground">{user.email}</strong>
      </p>
      {verificationUrl && (
        <Alert variant="success">
          <p className="mb-2 text-sm">Email is not configured on this server. Use this link to verify:</p>
          <a
            href={verificationUrl}
            className="break-all text-sm font-medium text-primary hover:underline"
          >
            {verificationUrl}
          </a>
        </Alert>
      )}
      {message && <Alert variant="success">{message}</Alert>}
      {error && <Alert variant="error">{error}</Alert>}
      <div className="mt-4 flex flex-col gap-2">
        <Button onClick={handleResend} disabled={sending} className="w-full">
          {sending ? "Sending..." : "Resend verification email"}
        </Button>
        <Button variant="ghost" onClick={logout} className="w-full">
          Sign out
        </Button>
      </div>
      <p className="mt-6 text-center text-sm text-muted">
        Wrong account?{" "}
        <Link href={redirect ? `/login?redirect=${encodeURIComponent(redirect)}` : "/login"} className="font-medium text-primary hover:underline">
          Sign in
        </Link>
      </p>
    </AuthShell>
  );
}
