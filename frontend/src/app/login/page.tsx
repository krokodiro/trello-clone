"use client";

import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useAuth } from "@/providers/auth-provider";
import { getRedirectPath, withRedirect } from "@/lib/redirect";
import { mutationError } from "@/lib/toast-utils";
import { Alert, AuthShell, Button, FieldLabel, Input } from "@/components/ui";
import { VerificationLinkAlert } from "@/components/verification-link-alert";
import { useToast } from "@/providers/toast-provider";

export default function LoginPage() {
  return (
    <Suspense
      fallback={
        <AuthShell title="Welcome back" subtitle="Sign in to your account">
          <p className="text-center text-sm text-muted">Loading...</p>
        </AuthShell>
      }
    >
      <LoginForm />
    </Suspense>
  );
}

function LoginForm() {
  const { login, resendVerification, verificationUrl } = useAuth();
  const { toast } = useToast();
  const router = useRouter();
  const searchParams = useSearchParams();
  const redirect = getRedirectPath(searchParams.toString());
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [unverified, setUnverified] = useState(false);
  const [resendMessage, setResendMessage] = useState("");
  const [loading, setLoading] = useState(false);
  const [resending, setResending] = useState(false);
  const [resetSuccess, setResetSuccess] = useState(false);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    setResetSuccess(params.get("reset") === "success");
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setUnverified(false);
    setResendMessage("");
    setLoading(true);
    try {
      await login(email, password);
      toast({ title: "Welcome back!", variant: "success" });
      router.push(redirect ?? "/");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Login failed";
      if (message === "email not verified") {
        setUnverified(true);
        setError("Please verify your email before signing in.");
      } else {
        setError(message);
        toast({ title: "Sign in failed", description: mutationError(err), variant: "error" });
      }
    } finally {
      setLoading(false);
    }
  };

  const handleResend = async () => {
    setResending(true);
    setResendMessage("");
    try {
      const url = await resendVerification(email);
      if (url) {
        setResendMessage("");
      } else {
        setResendMessage("Verification email sent. Check your inbox.");
      }
    } catch (err) {
      setResendMessage(err instanceof Error ? err.message : "Failed to send email");
    } finally {
      setResending(false);
    }
  };

  return (
    <AuthShell title="Welcome back" subtitle="Sign in to your account">
      <div className="space-y-4">
        {resetSuccess && (
          <Alert variant="success">
            Password updated. You can sign in with your new password.
          </Alert>
        )}
        <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <FieldLabel>Email</FieldLabel>
          <Input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
        </div>
        <div>
          <div className="mb-1.5 flex items-center justify-between">
            <FieldLabel className="mb-0">Password</FieldLabel>
            <Link href="/forgot-password" className="text-xs text-primary hover:underline">
              Forgot password?
            </Link>
          </div>
          <Input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </div>
        {error && <Alert variant="error">{error}</Alert>}
        {unverified && (
          <div className="space-y-2">
            <Button
              type="button"
              variant="secondary"
              className="w-full"
              onClick={handleResend}
              disabled={resending || !email}
            >
              {resending ? "Sending..." : "Resend verification email"}
            </Button>
            {resendMessage && <Alert variant="info">{resendMessage}</Alert>}
            {verificationUrl && <VerificationLinkAlert url={verificationUrl} />}
          </div>
        )}
        <Button type="submit" className="w-full" loading={loading}>
          Sign in
        </Button>
      </form>
      <p className="mt-6 text-center text-sm text-muted">
        No account?{" "}
        <Link href={withRedirect("/register", redirect)} className="font-medium text-primary hover:underline">
          Register
        </Link>
      </p>
      </div>
    </AuthShell>
  );
}
