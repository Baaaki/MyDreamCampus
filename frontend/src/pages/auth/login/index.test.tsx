import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import LoginPage from "./index";

function renderAt(initial = "/auth/login") {
  return render(
    <MemoryRouter initialEntries={[initial]}>
      <Routes>
        <Route path="/auth/login" element={<LoginPage />} />
        <Route path="/auth/change-password" element={<div>change-password-page</div>} />
        <Route path="/dashboard" element={<div>admin-home</div>} />
        <Route path="/teacher/attendance" element={<div>teacher-home</div>} />
        <Route path="/student/dashboard" element={<div>student-home</div>} />
        <Route path="/grades/transcripts" element={<div>safe-redirect-target</div>} />
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
  localStorage.clear();
  document.cookie = "csrf_token=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/";
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.resetAllMocks();
});

describe("LoginPage - happy paths", () => {
  it("redirects admin to /dashboard and stores user", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          access_token: "at",
          user: { id: "u-1", email: "a@x.tr", role: "admin" },
        })
      )
    );

    renderAt();
    await userEvent.type(screen.getByPlaceholderText("E-posta adresi"), "a@x.tr");
    await userEvent.type(screen.getByPlaceholderText("Şifre"), "Password1");
    await userEvent.click(screen.getByRole("button", { name: /giriş yap/i }));

    expect(await screen.findByText("admin-home")).toBeInTheDocument();
    expect(JSON.parse(localStorage.getItem("user")!)).toMatchObject({
      id: "u-1",
      role: "admin",
    });
  });

  it("redirects teacher to /teacher/attendance", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          access_token: "at",
          user: { id: "u-2", role: "teacher" },
        })
      )
    );

    renderAt();
    await userEvent.type(screen.getByPlaceholderText("E-posta adresi"), "t@x.tr");
    await userEvent.type(screen.getByPlaceholderText("Şifre"), "Password1");
    await userEvent.click(screen.getByRole("button", { name: /giriş yap/i }));

    expect(await screen.findByText("teacher-home")).toBeInTheDocument();
  });

  it("redirects student to /student/dashboard", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          access_token: "at",
          user: { id: "u-3", role: "student" },
        })
      )
    );

    renderAt();
    await userEvent.type(screen.getByPlaceholderText("E-posta adresi"), "s@x.tr");
    await userEvent.type(screen.getByPlaceholderText("Şifre"), "Password1");
    await userEvent.click(screen.getByRole("button", { name: /giriş yap/i }));

    expect(await screen.findByText("student-home")).toBeInTheDocument();
  });
});

describe("LoginPage - force_password_change", () => {
  it("redirects to /auth/change-password regardless of role", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          access_token: "at",
          user: { id: "u-1", role: "admin" },
          force_password_change: true,
        })
      )
    );

    renderAt();
    await userEvent.type(screen.getByPlaceholderText("E-posta adresi"), "a@x.tr");
    await userEvent.type(screen.getByPlaceholderText("Şifre"), "Password1");
    await userEvent.click(screen.getByRole("button", { name: /giriş yap/i }));

    expect(await screen.findByText("change-password-page")).toBeInTheDocument();
  });
});

describe("LoginPage - safe redirect param", () => {
  it("honors a safe relative redirect", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          access_token: "at",
          user: { id: "u-1", role: "admin" },
        })
      )
    );

    renderAt("/auth/login?redirect=/grades/transcripts");
    await userEvent.type(screen.getByPlaceholderText("E-posta adresi"), "a@x.tr");
    await userEvent.type(screen.getByPlaceholderText("Şifre"), "Password1");
    await userEvent.click(screen.getByRole("button", { name: /giriş yap/i }));

    expect(await screen.findByText("safe-redirect-target")).toBeInTheDocument();
  });

  it("ignores protocol-relative redirect (//evil.com)", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          access_token: "at",
          user: { id: "u-1", role: "admin" },
        })
      )
    );

    renderAt("/auth/login?redirect=//evil.com/x");
    await userEvent.type(screen.getByPlaceholderText("E-posta adresi"), "a@x.tr");
    await userEvent.type(screen.getByPlaceholderText("Şifre"), "Password1");
    await userEvent.click(screen.getByRole("button", { name: /giriş yap/i }));

    expect(await screen.findByText("admin-home")).toBeInTheDocument();
  });

  it("ignores absolute URL redirect (https://evil.com)", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          access_token: "at",
          user: { id: "u-1", role: "admin" },
        })
      )
    );

    renderAt("/auth/login?redirect=https://evil.com");
    await userEvent.type(screen.getByPlaceholderText("E-posta adresi"), "a@x.tr");
    await userEvent.type(screen.getByPlaceholderText("Şifre"), "Password1");
    await userEvent.click(screen.getByRole("button", { name: /giriş yap/i }));

    expect(await screen.findByText("admin-home")).toBeInTheDocument();
  });
});

describe("LoginPage - error handling", () => {
  it("shows error message on 401 and stays on page", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(JSON.stringify({ message: "invalid credentials" }), {
          status: 401,
          headers: { "Content-Type": "application/json" },
        })
      )
    );

    renderAt();
    await userEvent.type(screen.getByPlaceholderText("E-posta adresi"), "a@x.tr");
    await userEvent.type(screen.getByPlaceholderText("Şifre"), "wrong");
    await userEvent.click(screen.getByRole("button", { name: /giriş yap/i }));

    expect(
      await screen.findByText(/giriş başarısız|invalid|request failed/i)
    ).toBeInTheDocument();
    expect(screen.queryByText("admin-home")).not.toBeInTheDocument();
  });

  it("does not store user on failure", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(JSON.stringify({ message: "locked" }), { status: 423 })
      )
    );

    renderAt();
    await userEvent.type(screen.getByPlaceholderText("E-posta adresi"), "a@x.tr");
    await userEvent.type(screen.getByPlaceholderText("Şifre"), "x");
    await userEvent.click(screen.getByRole("button", { name: /giriş yap/i }));

    await screen.findByText(/giriş başarısız|locked|request failed/i);
    expect(localStorage.getItem("user")).toBeNull();
  });
});
