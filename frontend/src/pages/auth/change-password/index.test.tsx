import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import ChangePasswordPage from "./index";

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/auth/change-password"]}>
      <Routes>
        <Route path="/auth/change-password" element={<ChangePasswordPage />} />
        <Route path="/auth/login" element={<div>login-page</div>} />
      </Routes>
    </MemoryRouter>
  );
}

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

beforeEach(() => {
  localStorage.setItem("user", JSON.stringify({ id: "u-1", role: "student" }));
  document.cookie = "csrf_token=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/";
  vi.spyOn(window, "alert").mockImplementation(() => {});
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

async function fillForm(current: string, next: string, confirm: string) {
  await userEvent.type(screen.getByPlaceholderText(/mevcut şifreniz/i), current);
  await userEvent.type(screen.getByPlaceholderText(/yeni şifreniz \(min/i), next);
  await userEvent.type(screen.getByPlaceholderText(/yeni şifrenizi tekrar/i), confirm);
}

describe("ChangePasswordPage - client validation", () => {
  it("rejects when new and confirm do not match (no API call)", async () => {
    const fetchSpy = vi.fn(async () => jsonResponse({}));
    vi.stubGlobal("fetch", fetchSpy);

    renderPage();
    await fillForm("Old1word", "NewPass1", "OtherPass1");
    await userEvent.click(screen.getByRole("button", { name: /şifre değiştir/i }));

    expect(await screen.findByText(/eşleşmiyor/i)).toBeInTheDocument();
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it("rejects passwords that violate the policy (too short, no upper, etc.)", async () => {
    const fetchSpy = vi.fn(async () => jsonResponse({}));
    vi.stubGlobal("fetch", fetchSpy);

    renderPage();
    await fillForm("Old1word", "weakpass", "weakpass");
    await userEvent.click(screen.getByRole("button", { name: /şifre değiştir/i }));

    expect(
      await screen.findByText(/en az 8 karakter|buyuk harf|kucuk harf|rakam/i)
    ).toBeInTheDocument();
    expect(fetchSpy).not.toHaveBeenCalled();
  });
});

describe("ChangePasswordPage - happy path", () => {
  it("clears stored user and redirects to /auth/login on success", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse({ message: "ok" }))
    );

    renderPage();
    await fillForm("Old1word", "NewPass1", "NewPass1");
    await userEvent.click(screen.getByRole("button", { name: /şifre değiştir/i }));

    expect(await screen.findByText("login-page")).toBeInTheDocument();
    expect(localStorage.getItem("user")).toBeNull();
    expect(window.alert).toHaveBeenCalledWith(expect.stringMatching(/başarıyla/i));
  });

  it("posts old_password and new_password to the change-password endpoint", async () => {
    let capturedUrl = "";
    let capturedMethod = "";
    let capturedBody = "";
    const fetchSpy = vi.fn(async (input: Request) => {
      capturedUrl = input.url;
      capturedMethod = input.method;
      capturedBody = await input.text();
      return jsonResponse({ message: "ok" });
    });
    vi.stubGlobal("fetch", fetchSpy);

    renderPage();
    await fillForm("Old1word", "NewPass1", "NewPass1");
    await userEvent.click(screen.getByRole("button", { name: /şifre değiştir/i }));

    await screen.findByText("login-page");

    expect(capturedUrl).toMatch(/\/api\/auth\/change-password$/);
    expect(capturedMethod).toBe("POST");
    expect(JSON.parse(capturedBody)).toEqual({
      old_password: "Old1word",
      new_password: "NewPass1",
    });
  });
});

describe("ChangePasswordPage - server error", () => {
  it("shows error message on API failure and keeps stored user", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(JSON.stringify({ message: "wrong old password" }), {
          status: 400,
          headers: { "Content-Type": "application/json" },
        })
      )
    );

    renderPage();
    await fillForm("BadOld11", "NewPass1", "NewPass1");
    await userEvent.click(screen.getByRole("button", { name: /şifre değiştir/i }));

    expect(
      await screen.findByText(/şifre değiştirme başarısız|wrong old|request failed/i)
    ).toBeInTheDocument();
    expect(localStorage.getItem("user")).not.toBeNull();
  });
});
