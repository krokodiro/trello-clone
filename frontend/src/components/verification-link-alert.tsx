import { Alert } from "@/components/ui";

export function VerificationLinkAlert({ url }: { url: string }) {
  return (
    <Alert variant="success">
      <p className="mb-2 text-sm">
        Email is not configured on this server. Open this link to verify your account:
      </p>
      <a
        href={url}
        className="break-all text-sm font-medium text-primary hover:underline"
      >
        {url}
      </a>
    </Alert>
  );
}
