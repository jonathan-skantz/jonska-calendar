export const api = async <T>(
  path: string,
  options?: RequestInit,
): Promise<T> => {
  const response = await fetch(path, {
    credentials: "include",
    headers: { "Content-Type": "application/json", ...options?.headers },
    ...options,
  });
  if (!response.ok)
    throw new Error((await response.text()) || "Request failed");
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
};
