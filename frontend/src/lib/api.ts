export const api = async <T>(
  path: string,
  options?: RequestInit,
): Promise<T> => {
  const { headers, ...rest } = options ?? {};
  const response = await fetch(path, {
    credentials: "include",
    ...rest,
    headers: { "Content-Type": "application/json", ...headers },
  });
  if (!response.ok)
    throw new Error((await response.text()) || "Request failed");
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
};
