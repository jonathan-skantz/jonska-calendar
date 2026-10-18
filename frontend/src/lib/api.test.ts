import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "./api";

describe("api", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("keeps the JSON content type when the caller adds headers", async () => {
    const fetchMock = vi.fn(async () => new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);

    await api("/api/example", {
      method: "POST",
      headers: { "X-Test": "1" },
    });

    const [, init] = fetchMock.mock.calls[0] as unknown as [
      string,
      RequestInit,
    ];
    expect(new Headers(init.headers).get("Content-Type")).toBe(
      "application/json",
    );
    expect(new Headers(init.headers).get("X-Test")).toBe("1");
    expect(init.credentials).toBe("include");
  });
});
