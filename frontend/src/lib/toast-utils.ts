export function mutationError(err: unknown): string {
  return err instanceof Error ? err.message : "Something went wrong";
}
