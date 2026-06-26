import { Alert } from "@/components/ui";

export function AuthLinkAlert({
  url,
  description,
}: {
  url: string;
  description: string;
}) {
  return (
    <Alert variant="success">
      <p className="mb-2 text-sm">{description}</p>
      <a
        href={url}
        className="break-all text-sm font-medium text-primary hover:underline"
      >
        {url}
      </a>
    </Alert>
  );
}
