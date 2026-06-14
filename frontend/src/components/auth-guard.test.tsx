import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { AuthGuard } from "./auth-guard";

function renderAt(path: string, allowedRoles: string[]) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route element={<AuthGuard allowedRoles={allowedRoles} />}>
          <Route path={path} element={<div>protected</div>} />
        </Route>
        <Route path="/auth/login" element={<div>login</div>} />
        <Route path="/dashboard" element={<div>admin-home</div>} />
        <Route path="/teacher/attendance" element={<div>teacher-home</div>} />
        <Route path="/student/dashboard" element={<div>student-home</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe("AuthGuard", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("redirects to login when no user in storage", () => {
    renderAt("/admin", ["admin"]);
    expect(screen.getByText("login")).toBeInTheDocument();
  });

  it("redirects to login when stored user is corrupt JSON", () => {
    localStorage.setItem("user", "not-json{{");
    renderAt("/admin", ["admin"]);
    expect(screen.getByText("login")).toBeInTheDocument();
  });

  it("renders outlet when role is allowed", () => {
    localStorage.setItem("user", JSON.stringify({ id: "1", role: "admin" }));
    renderAt("/admin", ["admin"]);
    expect(screen.getByText("protected")).toBeInTheDocument();
  });

  it("redirects admin to /dashboard if role mismatches", () => {
    localStorage.setItem("user", JSON.stringify({ id: "1", role: "admin" }));
    renderAt("/student-page", ["student"]);
    expect(screen.getByText("admin-home")).toBeInTheDocument();
  });

  it("redirects teacher to /teacher/attendance if role mismatches", () => {
    localStorage.setItem("user", JSON.stringify({ id: "1", role: "teacher" }));
    renderAt("/admin", ["admin"]);
    expect(screen.getByText("teacher-home")).toBeInTheDocument();
  });

  it("redirects student to /student/dashboard if role mismatches", () => {
    localStorage.setItem("user", JSON.stringify({ id: "1", role: "student" }));
    renderAt("/admin", ["admin"]);
    expect(screen.getByText("student-home")).toBeInTheDocument();
  });

  it("redirects unknown role back to login", () => {
    localStorage.setItem("user", JSON.stringify({ id: "1", role: "ghost" }));
    renderAt("/admin", ["admin"]);
    expect(screen.getByText("login")).toBeInTheDocument();
  });
});
