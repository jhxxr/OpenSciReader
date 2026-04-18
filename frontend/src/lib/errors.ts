export function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error) {
    const message = error.message.trim();
    if (message) {
      return message;
    }
  }

  if (typeof error === "string") {
    const message = error.trim();
    if (message) {
      return message;
    }
  }

  if (typeof error === "object" && error !== null) {
    const record = error as Record<string, unknown>;
    for (const key of ["message", "error", "err"]) {
      const value = record[key];
      if (typeof value === "string" && value.trim()) {
        return value.trim();
      }
    }

    if ("cause" in record) {
      return getErrorMessage(record.cause, fallback);
    }
  }

  return fallback;
}
